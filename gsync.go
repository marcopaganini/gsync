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
	"os"
	"os/user"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/marcopaganini/gsync/logger"
	"github.com/marcopaganini/gsync/vfs/gdrive"
	"github.com/marcopaganini/gsync/vfs/local"
)

const (
	AUTH_CACHE_FILE  = ".gsync-token-cache.json"
	CREDENTIALS_FILE = ".gsync-credentials.json"

	// Flag defaults
	DEFAULT_OPT_VERBOSE = false
	DEFAULT_OPT_DRY_RUN = false
)

type cmdLineOpts struct {
	clientId     string
	clientSecret string
	code         string
	dryrun       bool
	exclude      string
	verbose      bool
}

var (
	// Command line Flags
	opt cmdLineOpts

	// Generic logging object
	log *logger.Logger
)

type GdriveCredentials struct {
	ClientId     string
	ClientSecret string
}

// VFS interface
type gsyncVfs interface {
	FileTree(string) ([]string, error)
	FileExists(string) (bool, error)
	IsDir(string) (bool, error)
	IsRegular(string) (bool, error)
	Mkdir(string) error
	Mtime(string) (time.Time, error)
	ReadFromFile(string) (io.Reader, error)
	SetMtime(string, time.Time) error
	Size(string) (int64, error)
	WriteToFile(string, io.Reader) error
}

// Directory pairs for sync post-processing of directories
type dirpair struct {
	src string
	dst string
}

// Retrieve the sources and destination from the command-line, performing basic sanity checking.
//
// Returns:
// 	[]string: source paths
// 	string: destination directory
// 	error
func getSourceDest() ([]string, string, error) {
	var srcpaths []string

	if flag.NArg() < 2 {
		return nil, "", fmt.Errorf("Must specify source and destination directories")
	}

	// All arguments but last are considered to be sources
	for ix := 0; ix < flag.NArg()-1; ix++ {
		srcpaths = append(srcpaths, flag.Arg(ix))
	}
	dst := flag.Arg(flag.NArg() - 1)

	return srcpaths, dst, nil
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

// Initializes a new GdriveVFS instance. This is a helper wrapper to gdrivefs.NewGdriveFileSystem.
// This function calls handleCredentials to load/save the token and act on the Oauth code, if needed.
//
// Returns:
//   gsyncVfs
//   error
func initGdriveVfs(clientId string, clientSecret string, code string) (gsyncVfs, error) {
	// Credentials and cache file
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	credfile := path.Join(usr.HomeDir, CREDENTIALS_FILE)
	cachefile := path.Join(usr.HomeDir, AUTH_CACHE_FILE)

	// Load/save credentials
	cred, err := handleCredentials(credfile, clientId, clientSecret)
	if err != nil {
		return nil, err
	}

	// Initialize virtual filesystems
	g, err := gdrivevfs.NewGdriveFileSystem(cred.ClientId, cred.ClientSecret, opt.code, cachefile)
	if err != nil {
		return nil, err
	}
	return g, nil
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
func needToCopy(srcvfs gsyncVfs, dstvfs gsyncVfs, srcpath string, dstpath string) (bool, error) {
	// If destination doesn't exist we need to copy
	exists, err := dstvfs.FileExists(dstpath)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	// If destination exists, we check mtimes truncated to the nearest second
	srcMtime, err := srcvfs.Mtime(srcpath)
	if err != nil {
		return false, err
	}
	dstMtime, err := dstvfs.Mtime(dstpath)
	if err != nil {
		return false, err
	}

	srcMtime = srcMtime.Truncate(time.Second)
	dstMtime = dstMtime.Truncate(time.Second)

	if srcMtime.After(dstMtime) {
		return true, nil
	}

	return false, nil
}

// Copy the content of all files/directories pointed by srcpath into dstdir.
// If srcpath is a file, the file will be copied. If it is a directory, the
// entire subtree will be copied.  Dstdir must be a directory.
//
// Like rsync, a source path ending in slash means "copy the contents of this
// directory into the destination" whereas a path not ending in a slash means
// "copy this directory and its contents into the destination."
//
// Files/directories are only copied if needed (based on the modification date
// of the file on both filesystems.) This function uses the srcvfs and dstvfs
// VFS objects to perform operations on the respective filesystems.
//
// Return:
// 	 error
func sync(srcpath string, dstdir string, srcvfs gsyncVfs, dstvfs gsyncVfs) error {
	var (
		srctree  []string
		dirpairs []dirpair
	)

	// Destination must exist and be a directory
	exists, err := dstvfs.FileExists(dstdir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Destination \"%s\" does not exist", dstdir)
	}

	isdir, err := dstvfs.IsDir(dstdir)
	if err != nil {
		return err
	}
	if !isdir {
		return fmt.Errorf("Destination \"%s\" is not a directory/folder", dstdir)
	}

	// Special case: If the source path is not a directory, we short circuit
	// the FileTree method here and set srctree to that single file.
	isdir, err = srcvfs.IsDir(srcpath)
	if err != nil {
		return err
	}
	if isdir {
		srctree, err = srcvfs.FileTree(srcpath)
		if err != nil {
			return err
		}
	} else {
		srctree = []string{srcpath}
	}

	// Guarantee that we'll process a directory before files inside it
	sort.Strings(srctree)

	for _, src := range srctree {
		// If the source path ends in a slash, we'll copy the *contents* of the
		// source directory to the destination. If it doesn't, we'll create a
		// directory inside the destination. This matches rsync's behavior
		//
		// Ex:
		// /a/b/c/ -> foo = /foo/<files>...
		// /a/b/c  -> foo = /foo/c/<files>...

		// Default == copy files INTO directory at destination
		dst := path.Join(dstdir, src[len(srcpath):])

		// If source does not end in "/", we create the directory specified
		// by srcpath as the first level inside the destination.
		if !strings.HasSuffix(srcpath, "/") {
			sdir := strings.Split(srcpath, "/")
			if len(sdir) > 1 {
				last := len(sdir) - 1
				ssrc := strings.Split(src, "/")
				dst = path.Join(dstdir, strings.Join(ssrc[last:], "/"))
			}
		}

		isdir, err := srcvfs.IsDir(src)
		if err != nil {
			return err
		}
		isregular, err := srcvfs.IsRegular(src)
		if err != nil {
			return err
		}

		// Start sync operation

		if isdir {
			// Create destination dir if needed
			exists, err := dstvfs.FileExists(dst)
			if err != nil {
				return err
			}
			if !exists {
				log.Verboseln(1, dst)
				if !opt.dryrun {
					err := dstvfs.Mkdir(dst)
					if err != nil {
						return err
					}
				}
			}
			// Save directory for post processing
			d := dirpair{src, dst}
			dirpairs = append(dirpairs, d)
		} else if isregular {
			copyNeeded, err := needToCopy(srcvfs, dstvfs, src, dst)
			if err != nil {
				return err
			}

			if copyNeeded {
				if !opt.dryrun {
					r, err := srcvfs.ReadFromFile(src)
					if err != nil {
						return err
					}
					err = dstvfs.WriteToFile(dst, r)
					if err != nil {
						return err
					}
					// Set destination mtime == source mtime
					mtime, err := srcvfs.Mtime(src)
					if err != nil {
						return err
					}
					err = dstvfs.SetMtime(dst, mtime)
					if err != nil {
						return err
					}
				}
				log.Verboseln(1, dst)
			}
		} else {
			log.Printf("Warning: Ignoring \"%s\" which is not a file or directory.\n", src)
			continue
		}
	}

	// Set the mtimes of all destination directories to the original mtimes.
	// We have to do it here (and bottom first!) because in certain filesystems,
	// updating files inside directories will also change the directory mtime.

	if !opt.dryrun {
		for ix := len(dirpairs) - 1; ix >= 0; ix-- {
			src := dirpairs[ix].src
			dst := dirpairs[ix].dst

			mtime, err := srcvfs.Mtime(src)
			if err != nil {
				return err
			}
			err = dstvfs.SetMtime(dst, mtime)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

func main() {
	var (
		srcvfs   gsyncVfs
		dstvfs   gsyncVfs
		gfs      gsyncVfs
		lfs      gsyncVfs
		srcdir   string
		dstdir   string
		srcpaths []string
	)

	// Parse command line
	flag.StringVar(&opt.clientId, "id", "", "Client ID")
	flag.StringVar(&opt.clientSecret, "secret", "", "Client Secret")
	flag.StringVar(&opt.code, "code", "", "Authorization Code")
	flag.BoolVar(&opt.dryrun, "dry-run", DEFAULT_OPT_DRY_RUN, "Dry-run mode")
	flag.BoolVar(&opt.dryrun, "n", DEFAULT_OPT_DRY_RUN, "Dry-run mode (shorthand)")
	flag.BoolVar(&opt.verbose, "verbose", DEFAULT_OPT_VERBOSE, "Verbose Mode")
	flag.BoolVar(&opt.verbose, "v", DEFAULT_OPT_VERBOSE, "Verbose mode (shorthand)")
	flag.Parse()

	// Set verbose level
	log = logger.New("")
	if opt.verbose {
		log.SetVerboseLevel(1)
	}

	srcpaths, dstdir, err := getSourceDest()
	if err != nil {
		usage(err)
	}

	// Initialize virtual filesystems
	gfs, err = initGdriveVfs(opt.clientId, opt.clientSecret, opt.code)
	if err != nil {
		log.Fatal(err)
	}
	lfs = localvfs.NewLocalFileSystem()
	dstvfs = lfs
	isDstGdrive, dstPath := isGdrivePath(dstdir)
	if isDstGdrive {
		dstvfs = gfs
	}

	// Treat each path separately
	for _, srcdir = range srcpaths {
		isSrcGdrive, srcPath := isGdrivePath(srcdir)

		// Select VFSes according to path type
		srcvfs = lfs
		if isSrcGdrive {
			srcvfs = gfs
		}

		// Sync
		err = sync(srcPath, dstPath, srcvfs, dstvfs)
		if err != nil {
			log.Fatal(err)
		}
	}
}
