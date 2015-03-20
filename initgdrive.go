package main

// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/user"
	"path"

	"github.com/marcopaganini/gsync/vfs/gdrive"
)

const (
	authCacheFile   = ".gsync-token-cache.json"
	credentialsFile = ".gsync-credentials.json"
)

// GdriveCredentials contain the ClientID & secret credentials for Google Drive.
type GdriveCredentials struct {
	ClientID     string
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
func handleCredentials(credFile string, clientID string, clientSecret string) (*GdriveCredentials, error) {
	var cred *GdriveCredentials

	// If client, secret and code specified, save config
	if clientID != "" && clientSecret != "" {
		cred = &GdriveCredentials{ClientID: clientID, ClientSecret: clientSecret}
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
func initGdriveVfs(clientID string, clientSecret string, code string) (gsyncVfs, error) {
	// Credentials and cache file
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	credfile := path.Join(usr.HomeDir, credentialsFile)
	cachefile := path.Join(usr.HomeDir, authCacheFile)

	// Load/save credentials
	cred, err := handleCredentials(credfile, clientID, clientSecret)
	if err != nil {
		return nil, err
	}

	// Initialize virtual filesystems
	g, err := gdrivevfs.NewGdriveFileSystem(cred.ClientID, cred.ClientSecret, opt.code, cachefile)
	if err != nil {
		return nil, err
	}
	return g, nil
}
