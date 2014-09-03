package localvfs

// Local filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"
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

// TODO: Implement these
func (gfs *localFileSystem) WriteToFile(dst string, reader io.Reader) error {
	return fmt.Errorf("Not implemented")
}

func (gfs *localFileSystem) ReadFromFile(fullpath string, writer io.Writer) (int64, error) {
	return 0, fmt.Errorf("Not implemented")
}

func (gfs *localFileSystem) Mkdir(path string) error {
	return fmt.Errorf("Not implemented")
}
