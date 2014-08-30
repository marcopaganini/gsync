package fs

// Local filesystem abstractions for gsync
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"os"
	"path/filepath"
)

type localFile struct {
	path string
	fi   os.FileInfo
}

// Local drive filesystem representation
type localFileSystem struct {
	path     string
	fileList []*localFile
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
	err := filepath.Walk(fs.path, func(srcpath string, fi os.FileInfo, err error) error {
		lfile := &localFile{path: srcpath, fi: fi}
		fs.fileList = append(fs.fileList, lfile)
		return nil
	})
	return err
}

// Return an array containing a list representing file objects in the local
// filesystem.
//
// Returns:
//   []string - Array containing the list of files
//   error
func (fs *localFileSystem) FileTree() []*localFile {
	return fs.fileList
}

// Return the full name of the localFile object
func (fs *localFileSystem) FullName(lfile *localFile) string {
	return lfile.path
}

// Return true if localFile is a directory
func (fs *localFileSystem) IsDir(lfile *localFile) bool {
	return lfile.fi.IsDir()
}

// Return true if localFile is a Regular File
func (fs *localFileSystem) IsRegular(lfile *localFile) bool {
	return lfile.fi.Mode().IsRegular()
}

// Return the file name of the localFile object
func (fs *localFileSystem) Name(lfile *localFile) string {
	return lfile.fi.Name()
}

// Return the size of the localFile object
func (fs *localFileSystem) Size(lfile *localFile) int64 {
	return lfile.fi.Size()
}
