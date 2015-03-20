package localvfs

// Local filesystem abstractions for gsync
//
// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// LocalFileSystem holds state on an instance of LocalFileSystem.
type LocalFileSystem struct {
	optWriteInPlace bool
}

// NewLocalFileSystem creates a new LocalFileSystem object
func NewLocalFileSystem() *LocalFileSystem {
	fs := &LocalFileSystem{}
	return fs
}

// FileExists returns true if a file/directory exists. False otherwise.
func (fs *LocalFileSystem) FileExists(fullpath string) (bool, error) {
	_, err := os.Stat(fullpath)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// FileTree returns a slice containing all files/directories under fullpath.
func (fs *LocalFileSystem) FileTree(fullpath string) ([]string, error) {
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
	for k := range pathMap {
		pathSlice = append(pathSlice, k)
	}
	sort.Strings(pathSlice)
	return pathSlice, nil
}

// IsDir returns true if fullpath is a directory, false if it isn't or if the
// file doesn't exist.
func (fs *LocalFileSystem) IsDir(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fi.Mode().IsDir(), nil
}

// IsRegular returns true if fullpath is a regular file, false if it isn't or
// if the file doesn't exist.
func (fs *LocalFileSystem) IsRegular(fullpath string) (bool, error) {
	fi, err := os.Stat(fullpath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

// Mkdir creates a directory named 'path'
func (fs *LocalFileSystem) Mkdir(path string) error {
	err := os.Mkdir(path, 0755)
	return err
}

// Mtime returns the local file's Modified Time (mtime) truncated to the
// nearest second (no nano information).
func (fs *LocalFileSystem) Mtime(fullpath string) (time.Time, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

// ReadFromFile returns an io.Reader pointing to fullpath in the local filesystem.
func (fs *LocalFileSystem) ReadFromFile(fullpath string) (io.Reader, error) {
	return os.Open(fullpath)
}

// SetMtime sets the 'modification time' of fullpath to mtime
func (fs *LocalFileSystem) SetMtime(fullpath string, mtime time.Time) error {
	atime := time.Now()
	return os.Chtimes(fullpath, atime, mtime)
}

// SetWriteInPlace sets the 'write in place' option. This will cause write operations
// to not use an intermediate temporary file and an atomic rename.
func (fs *LocalFileSystem) SetWriteInPlace(f bool) {
	fs.optWriteInPlace = f
}

// Size returns the size of the file pointed by fullpath, in bytes.
func (fs *LocalFileSystem) Size(fullpath string) (int64, error) {
	fi, err := os.Stat(fullpath)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// WriteToFile reads all data from reader and write to file fullpath.
func (fs *LocalFileSystem) WriteToFile(fullpath string, reader io.Reader) error {
	var (
		outWriter *os.File
		tmpFile   string
	)

	dir := filepath.Dir(fullpath)
	name := filepath.Base(fullpath)

	if name == "" {
		return fmt.Errorf("Trying to write to empty name")
	}

	// If the file exists, it must be a regular file
	// We don't support writing to directories.
	fi, err := os.Stat(fullpath)
	if err != nil {
		if os.IsExist(err) && !fi.Mode().IsRegular() {
			return fmt.Errorf("Local path \"%s\" exists and is not a regular file", fullpath)
		}
	}

	if fs.optWriteInPlace {
		os.Remove(fullpath)
		outWriter, err = os.Create(fullpath)
		if err != nil {
			return err
		}
		defer outWriter.Close()
	} else {
		// Create a temporary file and write to it, renaming at the end.
		outWriter, err = ioutil.TempFile(dir, name)
		if err != nil {
			return err
		}
		tmpFile = outWriter.Name()
		defer outWriter.Close()
		defer os.Remove(tmpFile)
	}

	_, err = io.Copy(outWriter, reader)
	if err != nil {
		return err
	}
	outWriter.Close()

	if !fs.optWriteInPlace {
		err = os.Rename(tmpFile, fullpath)
		if err != nil {
			return err
		}
	}

	return nil
}
