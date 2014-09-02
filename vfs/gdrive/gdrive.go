package gdrivevfs

// Gdrive filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"

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
	fileList     []*gdriveFile
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

// Return the base path for this virtual filesystem.
//
// Returns:
// 	 string
func (gfs *gdriveFileSystem) Path() string {
	return gfs.path
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

// TODO: Implement these
func (gfs *gdriveFileSystem) WriteToFile(dst string, reader io.Reader) error {
	return fmt.Errorf("Not implemented")
}

func (gfs *gdriveFileSystem) PathOutdated(src string, dst string) (bool, error) {
	return false, fmt.Errorf("Not implemented")
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
