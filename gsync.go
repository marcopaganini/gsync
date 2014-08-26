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
	"os/user"
	"path"

	drive "code.google.com/p/google-api-go-client/drive/v2"
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
	requestURL   = flag.String("request_url", "https://www.googleapis.com/oauth2/v1/userinfo", "API request")
)

type GdriveCredentials struct {
	ClientId     string
	ClientSecret string
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

func main() {
	//var dirs []string

	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to get user information")
	}

	credfile := path.Join(usr.HomeDir, CREDENTIALS_FILE)
	cred, err := handleCredentials(credfile, *clientId, *clientSecret)
	if err != nil {
		log.Fatal(err)
	}

	cachefile := path.Join(usr.HomeDir, AUTH_CACHE_FILE)
	g, err := gdp.NewGdrivePath(cred.ClientId, cred.ClientSecret, *code, drive.DriveScope, cachefile)
	if err != nil {
		log.Fatalf("Unable to initialize GdrivePath: %v", err)
	}

	/*
		_, err = g.Insert("tmp/foofile", "/tmp/foofile")
		if err != nil {
			log.Fatalln(err)
		}
	*/

	err = g.Download("tmp/foofile", "/tmp/foofile2")
	if err != nil {
		log.Fatalln(err)
	}

	/*
		// Copy parameters
		dstDir := "pix/.thumbnails"
		srcdir := "/home/paganini/pix/.thumbnails"

		dirs = append(dirs, srcdir)

		idx := 0
		for idx < len(dirs) {
			dirname := dirs[idx]
			subDir := path.Join(dstDir, dirname[len(srcdir):])

			fmt.Printf("====> %s\n", dirname)
			fmt.Printf("Creating %s\n", subDir)

			// Create destination dir
			dstObj, err := g.Mkdir(subDir)
			if err != nil {
				log.Fatalf("Mkdir(%s): %v", subDir, err)
			}
			if dstObj == nil {
				log.Fatalf("Mkdir(%s): Unable to create", subDir)
			}

			flist, err := io.ReadDir(dirname)
			if err != nil {
				log.Fatalf("ReadDir(%s): %v", dirname, err)
			}

			for _, fi := range flist {
				if fi.Mode().IsDir() {
					dirs = append(dirs, path.Join(dirname, fi.Name()))
				}
				if fi.Mode().IsRegular() {
					localFile := path.Join(dirname, fi.Name())
					remoteFile := path.Join(subDir, fi.Name())

					copyStat := "Not copied"

					copyNeeded, err := g.RemotePathOutdated(remoteFile, localFile)
					if err != nil {
						log.Fatalln(err)
					}

					if copyNeeded {
						copyStat = "Copied"
						driveFile, err := g.Insert(remoteFile, localFile)
						if err != nil {
							log.Fatalf("Insert(%s->%s): %v", localFile, remoteFile, err)
						}
						if driveFile == nil {
							log.Fatalf("Insert(%s->%s): Error in path inserting file", localFile, remoteFile)
						}
					}
					fmt.Printf("    %8d %s [%s]\n", fi.Size(), localFile, copyStat)
				}
			}
			idx++
		}
	*/
}
