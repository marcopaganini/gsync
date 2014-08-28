package main

// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"code.google.com/p/google-api-go-client/drive/v2"
	gdp "github.com/marcopaganini/gdrive_path"
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

	// Only supports copies *to* Google Drive for now
	if strings.HasPrefix(src, "g:") || strings.HasPrefix(src, "gdrive:") ||
		!(strings.HasPrefix(dst, "g:") || strings.HasPrefix(dst, "gdrive:")) {
		return "", "", fmt.Errorf("Temporarily, only copies to Google Drive are supported")
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

// Initialize Gdrive using the gdrive_path library. Uses CREDENTIALS_FILE and AUTH_CACHE_FILE
// under the current user's homedir to store credentials and the token, respectively.
//
// Returns:
//   *gdrive.Gdrive
//   error
func initGdrive() (*gdp.Gdrive, error) {
	// Create Gdrive object & authenticate
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("Unable to get current user information from the OS")
	}

	credfile := path.Join(usr.HomeDir, CREDENTIALS_FILE)
	cred, err := handleCredentials(credfile, *clientId, *clientSecret)
	if err != nil {
		return nil, err
	}

	cachefile := path.Join(usr.HomeDir, AUTH_CACHE_FILE)
	g, err := gdp.NewGdrivePath(cred.ClientId, cred.ClientSecret, *code, drive.DriveScope, cachefile)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize GdrivePath: %v", err)
	}

	return g, nil
}

// Prints error message and program usage to stderr, exit the program.
func usage(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	fmt.Fprintf(os.Stderr, "Usage%s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	//var dirs []string

	flag.Parse()

	srcdir, dstdir, err := getSourceDest()
	if err != nil {
		usage(err)
	}
	// TODO:For now we just remove the g: or gdrive: prefixes dstdir
	idx := strings.Index(dstdir, ":")
	if idx != -1 {
		dstdir = dstdir[idx+1:]
	}

	g, err := initGdrive()

	err = filepath.Walk(srcdir, func(src string, fi os.FileInfo, err error) error {
		// We always copy from a directory *INTO* a destination directory
		// Similar to rsync's rsync source/ dest.
		// TODO(Fix this later)
		dst := path.Join(dstdir, src[len(srcdir):])

		// If directory, create remote
		if fi.IsDir() {
			fmt.Printf("====> %s\n", src)
			fmt.Printf("      %s\n", dst)

			// Create destination dir
			_, err := g.Mkdir(dst)
			if err != nil {
				log.Fatalln(err)
			}
		} else if fi.Mode().IsRegular() {
			copyStat := "Not copied"

			//fmt.Printf("Attempting to copy [%s] to [%s]\n", src, dst)
			copyNeeded, err := g.RemotePathOutdated(dst, src)
			if err != nil {
				log.Fatalln(err)
			}

			if copyNeeded {
				copyStat = "Copied"
				_, err := g.Insert(dst, src)
				if err != nil {
					log.Fatalln(err)
				}
			}
			fmt.Printf("    %8d %s -> %s [%s]\n", fi.Size(), src, dst, copyStat)
		} else {
			fmt.Printf("Warning: Ignoring \"%s\" which is not a file or directory.\n", src)
		}

		return nil
	})
}
