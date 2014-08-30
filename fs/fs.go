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
func NewLocalFileSystem(path string) *localFileSystem {
	fs := &localFileSystem{path: path}
	return fs
}

// Return an array containing a list representing file objects in the local
// filesystem.
//
// Returns:
//   []string - Array containing the list of files
//   error
func (fs *localFileSystem) FileTree() ([]*localFile, error) {

	if len(fs.fileList) == 0 {
		err := filepath.Walk(fs.path, func(srcpath string, fi os.FileInfo, err error) error {
			lfile := &localFile{path: srcpath, fi: fi}
			fs.fileList = append(fs.fileList, lfile)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return fs.fileList, nil
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
