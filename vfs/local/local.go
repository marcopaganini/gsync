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
}

// Create a new localFileSystem object
//
// Returns:
//   *localFileSystem
//   error
func NewLocalFileSystem() *localFileSystem {
	fs := &localFileSystem{}
	return fs
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

// Return a slice containing all files/directories under fullpath
//
// Returns:
//   []string - slice of full pathnames
//   error
func (fs *localFileSystem) FileTree(fullpath string) ([]string, error) {
	// Use a map so duplicates are removed automatically
	pathMap := make(map[string]bool)
	err := filepath.Walk(fullpath, func(srcpath string, _ os.FileInfo, err error) error {
		pathMap[srcpath] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Create sorted list
	pathSlice := []string{}
	for k, _ := range pathMap {
		pathSlice = append(pathSlice, k)
	}
	sort.Strings(pathSlice)
	return pathSlice, nil
}

// Return true if fullpath is a directory, false if it isn't or
// if the file doesn't exist.
//
// Returns:
// 	bool
//  error
func (fs *localFileSystem) IsDir(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fi.Mode().IsDir(), nil
}

// Return true if fullpath is a regular file, false if it isn't or
// if the file doesn't exist.
//
// Returns:
// 	bool
//  error
func (fs *localFileSystem) IsRegular(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		return false, nil
	}
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
	err := os.Mkdir(path, 0755)
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
