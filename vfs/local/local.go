package localvfs

// Local filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Local drive filesystem representation
type localFileSystem struct {
	path      string
	pathMap   map[string]bool
	pathSlice []string
}

// Create a new localFileSystem object
//
// Returns:
//   *localFileSystem
//   error
func NewLocalFileSystem(path string) (*localFileSystem, error) {
	fs := &localFileSystem{path: path}
	err := fs.init()
	return fs, err
}

// Initialize a localFileSystem object, loading the entire file tree under fs.path
//
// Returns:
//   error
func (fs *localFileSystem) init() error {
	fs.pathMap = make(map[string]bool)
	err := filepath.Walk(fs.path, func(srcpath string, _ os.FileInfo, err error) error {
		fs.pathMap[srcpath] = true
		return nil
	})
	if err != nil {
		return err
	}

	// Create sorted list
	fs.pathSlice = []string{}
	for k, _ := range fs.pathMap {
		fs.pathSlice = append(fs.pathSlice, k)
	}
	sort.Strings(fs.pathSlice)

	return nil
}

// Returns true if a file/directory exists. False otherwise.
//
// Returns:
//   bool
//   error
func (fs *localFileSystem) FileExists(fullpath string) (bool, error) {
	_, err := os.Stat(fullpath)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Return a slice containing all files/directories under 'path'
//
// Returns:
//   []string - slice of full pathnames
//   error
func (fs *localFileSystem) FileTree() ([]string, error) {
	return fs.pathSlice, nil
}

// Return true if fullpath is a directory, false otherwise.
//
// Returns:
// 	bool
//  error
func (fs *localFileSystem) IsDir(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return false, err
	}
	return fi.Mode().IsDir(), nil
}

// Return true if fullpath is a regular file, false otherwise.
//
// Returns:
// 	bool
//  error
func (fs *localFileSystem) IsRegular(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

// Create a local directory named 'path'
//
// Returns
//   error
func (fs *localFileSystem) Mkdir(path string) error {
	err := os.Mkdir(path, 0644)
	return err
}

// Return the local file's Modified Time (mtime) truncated to the nearest
// second (no nano information).
//
// Returns:
//   int64
//   error
func (fs *localFileSystem) Mtime(fullpath string) (time.Time, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime().Truncate(time.Second), nil
}

// Return the base path for this virtual filesystem.
//
// Returns:
// 	 string
func (fs *localFileSystem) Path() string {
	return fs.path
}

// Return an io.Reader pointing to fullpath in the local filesystem.
//
// Returns:
//   io.Reader
//   error
func (gfs *localFileSystem) ReadFromFile(fullpath string) (io.Reader, error) {
	return os.Open(fullpath)
}

// Return the size of fullpath, in bytes
//
// Returns:
// 	int64
//  error
func (fs *localFileSystem) Size(fullpath string) (int64, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// Read all data from reader and write to file fullpath
//
// Returns:
//   error
func (gfs *localFileSystem) WriteToFile(fullpath string, reader io.Reader) error {
	dir := filepath.Dir(fullpath)
	name := filepath.Base(fullpath)

	if name == "" {
		return fmt.Errorf("Trying to write to empty name")
	}

	// If the file exists, it must be a regular file
	fi, err := os.Stat(fullpath)
	if err != nil {
		if os.IsExist(err) && !fi.Mode().IsRegular() {
			return fmt.Errorf("Local path \"%s\" exists and is not a regular file", fullpath)
		}
	}

	// Create a temporary file and write to it, renaming at the end.
	tmpWriter, err := ioutil.TempFile(dir, name)
	if err != nil {
		return err
	}
	tmpFile := tmpWriter.Name()
	defer tmpWriter.Close()
	defer os.Remove(tmpFile)

	_, err = io.Copy(tmpWriter, reader)
	if err != nil {
		return err
	}
	tmpWriter.Close()

	err = os.Rename(tmpFile, fullpath)
	if err != nil {
		return err
	}

	return nil
}
