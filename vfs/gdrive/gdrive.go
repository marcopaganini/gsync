package gdrivevfs

// Gdrive filesystem abstractions for gsync
//
// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"time"

	"code.google.com/p/google-api-go-client/drive/v2"
	gdp "github.com/marcopaganini/gdrive_path"
)

// GdriveFileSystem represents a virtual filesystem in Google Drive.
type GdriveFileSystem struct {
	g            *gdp.Gdrive
	clientID     string
	clientSecret string
	cachefile    string
	code         string
	fileSlice    []string

	// Options
	optWriteInPlace bool
}

// NewGdriveFileSystem creates a new GdriveFileSystem object
func NewGdriveFileSystem(clientID string, clientSecret string, code string, cachefile string) (*GdriveFileSystem, error) {
	gfs := &GdriveFileSystem{
		clientID:     clientID,
		clientSecret: clientSecret,
		code:         code,
		cachefile:    cachefile}

	err := gfs.init()
	return gfs, err
}

// Initialize a GdriveFileSystem object, loading the entire file tree under path
func (gfs *GdriveFileSystem) init() error {
	var err error

	// Initialize GdrivePath
	gfs.g, err = gdp.NewGdrivePath(gfs.clientID, gfs.clientSecret, gfs.code, drive.DriveScope, gfs.cachefile)
	if err != nil {
		return fmt.Errorf("Unable to initialize GdrivePath: %v", err)
	}

	return nil
}

// FileExists returns true if a file/directory exists. False otherwise.
func (gfs *GdriveFileSystem) FileExists(fullpath string) (bool, error) {
	_, err := gfs.g.Stat(fullpath)
	// Only return error on a real error condition. For file not found, return
	// false, nil. This makes it easier for the caller to test for real errors.
	if err != nil {
		if gdp.IsObjectNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// FileTree returns a slice containing all files/directories under fullpath.
func (gfs *GdriveFileSystem) FileTree(fullpath string) ([]string, error) {
	// sanitize
	_, _, pathname := splitPath(fullpath)

	// We iterate over all objects inside 'pathname'. If they're a
	// directory, we append them to dirs. The loop below will finish
	// when no more directories to be processed exist.

	dirs := []string{pathname}
	idx := 0

	for idx < len(dirs) {
		dir := dirs[idx]

		flist, err := gfs.g.ListDir(dir, "")
		if err != nil {
			return nil, err
		}

		for _, driveFile := range flist {
			fullpath := filepath.Join(dir, driveFile.Title)
			gfs.fileSlice = append(gfs.fileSlice, fullpath)
			// Append to the list of dirs to process if directory
			if gdp.IsDir(driveFile) {
				dirs = append(dirs, fullpath)
			}
		}
		idx++
	}

	// Create sorted list so dirs appear before files inside them.
	sort.Strings(gfs.fileSlice)
	return gfs.fileSlice, nil

}

// IsDir returns true if fullpath is a directory, false if it isn't or if the
// file doesn't exist.
func (gfs *GdriveFileSystem) IsDir(fullpath string) (bool, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return false, err
	}
	return gdp.IsDir(driveFile), nil
}

// IsRegular returns true if fullpath is a regular file, false if it isn't or
// if the file doesn't exist.
func (gfs *GdriveFileSystem) IsRegular(fullpath string) (bool, error) {
	isdir, err := gfs.IsDir(fullpath)
	return !isdir, err
}

// Mkdir creates a directory named 'path'
func (gfs *GdriveFileSystem) Mkdir(path string) error {
	_, err := gfs.g.Mkdir(path)
	return err
}

// Mtime returns the local file's Modified Time (mtime) truncated to the
// nearest second (no nano information).
func (gfs *GdriveFileSystem) Mtime(fullpath string) (time.Time, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return time.Time{}, err
	}
	return gdp.ModifiedDate(driveFile)
}

// ReadFromFile returns an io.Reader pointing to fullpath in the local filesystem.
func (gfs *GdriveFileSystem) ReadFromFile(fullpath string) (io.Reader, error) {
	return gfs.g.Download(fullpath)
}

// SetMtime sets the 'modification time' of fullpath to mtime
func (gfs *GdriveFileSystem) SetMtime(fullpath string, mtime time.Time) error {
	_, err := gfs.g.SetModifiedDate(fullpath, mtime)
	return err
}

// SetWriteInPlace sets the 'write in place' option. This will cause write operations
// to not use an intermediate temporary file and an atomic rename.
func (gfs *GdriveFileSystem) SetWriteInPlace(f bool) {
	gfs.optWriteInPlace = f
}

// Size returns the size of the file pointed by fullpath, in bytes.
func (gfs *GdriveFileSystem) Size(fullpath string) (int64, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return 0, err
	}
	return driveFile.FileSize, nil
}

// WriteToFile reads all data from reader and write to file fullpath.
func (gfs *GdriveFileSystem) WriteToFile(fullpath string, reader io.Reader) error {
	var err error

	if gfs.optWriteInPlace {
		_, err = gfs.g.InsertInPlace(fullpath, reader)
	} else {
		_, err = gfs.g.Insert(fullpath, reader)
	}
	return err
}

// splitPath takes a Unix like pathname, splits it on its components, and
// remove empty elements and unnecessary leading and trailing slashes. It
// returns three elements: A string containing the directory, a string
// containing the file, and a string with the entire path, sanitized.
func splitPath(pathName string) (string, string, string) {
	var ret []string

	for _, e := range strings.Split(pathName, "/") {
		if e != "" {
			ret = append(ret, e)
		}
	}
	if len(ret) == 0 {
		return "", "", ""
	}
	if len(ret) == 1 {
		return "", ret[0], ret[0]
	}
	return strings.Join(ret[0:len(ret)-1], "/"), ret[len(ret)-1], strings.Join(ret, "/")
}
