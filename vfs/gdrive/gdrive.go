package gdrivevfs

// Gdrive filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

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

// Gdrive filesystem representation
type gdriveFileSystem struct {
	g            *gdp.Gdrive
	clientId     string
	clientSecret string
	cachefile    string
	code         string
	fileSlice    []string
}

// Create a new GdriveFileSystem object
//
// Returns:
//   *GdriveFileSystem
//   error
func NewGdriveFileSystem(clientId string, clientSecret string, code string, cachefile string) (*gdriveFileSystem, error) {
	gfs := &gdriveFileSystem{
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

// Return a slice of strings with the full filenames found on Google drive
// under 'fullpath'
//
// Returns:
//   []string
//   error
func (gfs *gdriveFileSystem) FileTree(fullpath string) ([]string, error) {
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

// Return an io.Reader pointing to fullpath inside Google Drive.
//
// Returns:
//   io.Reader
// 	 error
func (gfs *gdriveFileSystem) ReadFromFile(fullpath string) (io.Reader, error) {
	return gfs.g.Download(fullpath)
}

// Set the 'modification time' of fullpath to mtime
//
// Returns:
//   error
func (gfs *gdriveFileSystem) SetMtime(fullpath string, mtime time.Time) error {
	_, err := gfs.g.SetModifiedDate(fullpath, mtime)
	return err
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

// splitPath takes a Unix like pathname, splits it on its components, and
// remove empty elements and unnecessary leading and trailing slashes.
//
// Returns:
//   - string: directory
//   - string: filename
//   - string: completely reconstructed path.
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
