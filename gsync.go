package main

// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/marcopaganini/gsync/vfs/gdrive"
	"github.com/marcopaganini/gsync/vfs/local"
)

const (
	AUTH_CACHE_FILE  = ".gsync-token-cache.json"
	CREDENTIALS_FILE = ".gsync-credentials.json"
)

var (
	clientId     = flag.String("id", "", "Client ID")
	clientSecret = flag.String("secret", "", "Client Secret")
	code         = flag.String("code", "", "Authorization Code")
)

type GdriveCredentials struct {
	ClientId     string
	ClientSecret string
}

// Retrieve the source and destination from the command-line, performing basic sanity checking
//
// Returns:
// 	string: source directory
// 	string: destination directory
// 	error
func getSourceDest() (string, string, error) {
	if flag.NArg() != 2 {
		return "", "", fmt.Errorf("Must specify source and destination directories")
	}

	src := flag.Arg(0)
	dst := flag.Arg(1)

	// Only supports copies from local to Gdrive or vice versa.
	// Local->Local and Remote->Remote are not supported.
	srcGdrive, _ := isGdrivePath(src)
	dstGdrive, _ := isGdrivePath(dst)

	if (srcGdrive && dstGdrive) || (!srcGdrive && !dstGdrive) {
		return "", "", fmt.Errorf("Local/Local and Remote/Remote copies not supported.")
	}
	return src, dst, nil
}

// Save and/or load credentials from disk.
//
// If clientId and clientSecret are set, this function saves them to credFile. If not,
// it loads those parameters from disk and returns them.
//
// Returns:
//   - *GdriveCredentials: containing the credentials.
//   - error
func handleCredentials(credFile string, clientId string, clientSecret string) (*GdriveCredentials, error) {
	var cred *GdriveCredentials

	// If client, secret and code specified, save config
	if clientId != "" && clientSecret != "" {
		cred = &GdriveCredentials{ClientId: clientId, ClientSecret: clientSecret}
		j, err := json.Marshal(*cred)
		if err != nil {
			return nil, fmt.Errorf("Unable to convert configuration to JSON: %v", err)
		}

		if ioutil.WriteFile(credFile, j, 0600) != nil {
			return nil, fmt.Errorf("Unable to write configuration file \"%s\": %v", credFile, err)
		}
	} else {
		j, err := ioutil.ReadFile(credFile)
		if err != nil {
			return nil, fmt.Errorf("Unable to read configuration from \"%s\": %v", credFile, err)
		}
		cred = &GdriveCredentials{}
		if json.Unmarshal(j, cred) != nil {
			return nil, fmt.Errorf("Unable to decode configuration form \"%s\": %v", credFile, err)
		}
	}
	return cred, nil
}

// Prints error message and program usage to stderr, exit the program.
func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	}
	fmt.Fprintf(os.Stderr, "Usage%s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

type GsyncVfs interface {
	FileTree() ([]string, error)
	FileExists(string) (bool, error)
	IsDir(string) (bool, error)
	IsRegular(string) (bool, error)
	Mkdir(string) error
	Mtime(string) (time.Time, error)
	Path() string
	ReadFromFile(string) (io.Reader, error)
	Size(string) (int64, error)
	WriteToFile(string, io.Reader) error
}

// Check if fullpath looks like a gdrive path (starting with g: or gdrive:). If
// so, return true and the path without the prefix. Otherwise, return false and
// the path itself.
//
// Returns
//   bool
//   realpath
func isGdrivePath(fullpath string) (bool, string) {
	if strings.HasPrefix(fullpath, "g:") || strings.HasPrefix(fullpath, "gdrive:") {
		idx := strings.Index(fullpath, ":")
		return true, fullpath[idx+1:]
	}
	return false, fullpath
}

// Determine if we need to copy the file pointed by srcpath in srcvfs to
// the file dstpath in dstvfs.
//
// Return:
// 	 bool
// 	 error
func needToCopy(srcvfs GsyncVfs, dstvfs GsyncVfs, srcpath string, dstpath string) (bool, error) {
	// If destination doesn't exist we need to copy
	exists, err := dstvfs.FileExists(dstpath)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	// If destination exists, we check mtimes
	srcMtime, err := srcvfs.Mtime(srcpath)
	if err != nil {
		return false, err
	}
	dstMtime, err := dstvfs.Mtime(dstpath)
	if err != nil {
		return false, err
	}
	if srcMtime.After(dstMtime) {
		return true, nil
	}

	return false, nil
}

func Sync(srcvfs GsyncVfs, dstvfs GsyncVfs) error {
	var (
		srcdir string
		dstdir string
	)

	srcdir = srcvfs.Path()
	dstdir = dstvfs.Path()

	srctree, err := srcvfs.FileTree()
	if err != nil {
		log.Fatal(err)
	}
	for _, src := range srctree {
		// If the source path ends in a slash, we'll copy the *contents* of the
		// source directory to the destination. If it doesn't, we'll create a
		// directory inside the destination. This matches rsync's behavior
		//
		// Ex:
		// /a/b/c/ -> foo = /foo/<files>...
		// /a/b/c  -> foo = /foo/c/<files>...

		// Default == copy files INTO directory at destination
		dst := path.Join(dstdir, src[len(srcdir):])

		// If source does not end in "/", we create the directory specified
		// by srcdir as the first level inside the destination.
		if !strings.HasSuffix(srcdir, "/") {
			sdir := strings.Split(srcdir, "/")
			if len(sdir) > 1 {
				last := len(sdir) - 1
				ssrc := strings.Split(src, "/")
				dst = path.Join(dstdir, strings.Join(ssrc[last:], "/"))
			}
		}

		isdir, err := srcvfs.IsDir(src)
		if err != nil {
			log.Fatal(err)
		}
		isregular, err := srcvfs.IsRegular(src)
		if err != nil {
			log.Fatal(err)
		}

		// Start sync operation

		if isdir {
			fmt.Printf("====> %s\n", src)
			fmt.Printf("      %s\n", dst)

			// Create destination dir
			err := dstvfs.Mkdir(dst)
			if err != nil {
				log.Fatalln(err)
			}
		} else if isregular {
			copyStat := "Not copied"
			copyNeeded, err := needToCopy(srcvfs, dstvfs, src, dst)
			if err != nil {
				log.Fatalln(err)
			}

			if copyNeeded {
				r, err := srcvfs.ReadFromFile(src)
				if err != nil {
					log.Fatalln(err)
				}
				err = dstvfs.WriteToFile(dst, r)
				if err != nil {
					log.Fatalln(err)
				}
				copyStat = "Copied"
			}
			// Print details
			size, err := srcvfs.Size(src)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("    %8d %s -> %s [%s]\n", size, src, dst, copyStat)
		} else {
			fmt.Printf("Warning: Ignoring \"%s\" which is not a file or directory.\n", src)
		}
	}

	return nil
}

func main() {
	var (
		srcvfs GsyncVfs
		dstvfs GsyncVfs
	)

	flag.Parse()

	srcdir, dstdir, err := getSourceDest()
	if err != nil {
		usage(err)
	}

	srcGdrive, srcPath := isGdrivePath(srcdir)
	_, dstPath := isGdrivePath(dstdir)

	// Credentials and cache file
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	credfile := path.Join(usr.HomeDir, CREDENTIALS_FILE)
	cred, err := handleCredentials(credfile, *clientId, *clientSecret)
	if err != nil {
		log.Fatal(err)
	}
	cachefile := path.Join(usr.HomeDir, AUTH_CACHE_FILE)

	// Initialize virtual filesystems
	if srcGdrive {
		srcvfs, err = gdrivevfs.NewGdriveFileSystem(srcPath, cred.ClientId, cred.ClientSecret, *code, cachefile)
		if err != nil {
			log.Fatal(err)
		}

		dstvfs, err = localvfs.NewLocalFileSystem(dstPath)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		srcvfs, err = localvfs.NewLocalFileSystem(srcPath)
		if err != nil {
			log.Fatal(err)
		}
		dstvfs, err = gdrivevfs.NewGdriveFileSystem(dstPath, cred.ClientId, cred.ClientSecret, *code, cachefile)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = Sync(srcvfs, dstvfs)
	if err != nil {
		log.Fatal(err)
	}
}
