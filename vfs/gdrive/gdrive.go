package gdrivevfs

// Gdrive filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"
	"time"

	"code.google.com/p/google-api-go-client/drive/v2"
	gdp "github.com/marcopaganini/gdrive_path"
)

type gdriveFile struct {
	path      string
	driveFile *drive.File
}

// Gdrive filesystem representation
type gdriveFileSystem struct {
	g            *gdp.Gdrive
	path         string
	clientId     string
	clientSecret string
	cachefile    string
	code         string
	fileList     []string
}

// Create a new GdriveFileSystem object
//
// Returns:
//   *GdriveFileSystem
//   error
func NewGdriveFileSystem(path string, clientId string, clientSecret string, code string, cachefile string) (*gdriveFileSystem, error) {
	gfs := &gdriveFileSystem{
		path:         path,
		clientId:     clientId,
		clientSecret: clientSecret,
		code:         code,
		cachefile:    cachefile}

	err := gfs.init()
	return gfs, err
}

// Initialize a gdriveFileSystem object, loading the entire file tree under path
//
// Returns:
//   error
func (gfs *gdriveFileSystem) init() error {
	var err error

	// Initialize GdrivePath
	gfs.g, err = gdp.NewGdrivePath(gfs.clientId, gfs.clientSecret, gfs.code, drive.DriveScope, gfs.cachefile)
	if err != nil {
		return fmt.Errorf("Unable to initialize GdrivePath: %v", err)
	}

	return nil
}

// Returns true if a file/directory exists. False otherwise.
//
// Returns:
//   bool
//   error
func (gfs *gdriveFileSystem) FileExists(fullpath string) (bool, error) {
	_, err := gfs.g.Stat(fullpath)
	// Only return error on a real error condition. For file not found, return
	// false, nil. This makes it easier for the caller to test for real errors.
	if err != nil {
		if gdp.IsObjectNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// Return a slice of strings with the full filenames found under 'path' in the
// Gdrive filesystem
//
// Returns:
//   []string
//   error
func (gfs *gdriveFileSystem) FileTree() ([]string, error) {
	return nil, fmt.Errorf("Not Implemented")
}

// Returns true if fullpath is a directory. False otherwise.
//
// Returns:
// 	 bool
// 	 error
func (gfs *gdriveFileSystem) IsDir(fullpath string) (bool, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return false, err
	}
	return gdp.IsDir(driveFile), nil
}

// Return true if fullpath is *NOT* a directory, false otherwise.  Google Drive
// does not support special files, so this should be the exact inverse of IsDir.
//
// Returns:
// 	 bool
// 	 error

func (gfs *gdriveFileSystem) IsRegular(fullpath string) (bool, error) {
	isdir, err := gfs.IsDir(fullpath)
	return !isdir, err
}

// Create a directory named 'path' on Google Drive
//
// Returns
//   error
func (gfs *gdriveFileSystem) Mkdir(path string) error {
	_, err := gfs.g.Mkdir(path)
	return err
}

// Return the Gdrive file's Modified Time (mtime) truncated to the nearest
// second (no nano information).
//
// Returns:
//   int64
//   error
func (gfs *gdriveFileSystem) Mtime(fullpath string) (time.Time, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return time.Time{}, err
	}
	return gdp.ModifiedDate(driveFile)
}

// Return the base path for this virtual filesystem.
//
// Returns:
// 	 string
func (gfs *gdriveFileSystem) Path() string {
	return gfs.path
}

// Return an io.Reader pointing to fullpath inside Google Drive.
//
// Returns:
//   io.Reader
// 	 error
func (gfs *gdriveFileSystem) ReadFromFile(fullpath string) (io.Reader, error) {
	return gfs.g.Download(fullpath)
}

// Return the size of fullpath in bytes.
//
// Returns:
//   int64
//   error
func (gfs *gdriveFileSystem) Size(fullpath string) (int64, error) {
	driveFile, err := gfs.g.Stat(fullpath)
	if err != nil {
		return 0, err
	}
	return driveFile.FileSize, nil
}

// Write to Gdrive file fullpath with content from reader
//
// Returns:
// 	 error
func (gfs *gdriveFileSystem) WriteToFile(fullpath string, reader io.Reader) error {
	_, err := gfs.g.Insert(fullpath, reader)
	return err
}
