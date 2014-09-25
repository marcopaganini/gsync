package main

// gsync - A google drive syncer in Go
//
// See instructions in the README.md file that accompanies this program.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"strings"

	"github.com/marcopaganini/gsync/vfs/local"
	"github.com/marcopaganini/logger"
)

var (
	// Generic logging object
	log *logger.Logger
)

// VFS interface
type gsyncVfs interface {
	FileTree(string) ([]string, error)
	FileExists(string) (bool, error)
	IsDir(string) (bool, error)
	IsRegular(string) (bool, error)
	Mkdir(string) error
	Mtime(string) (time.Time, error)
	ReadFromFile(string) (io.Reader, error)
	SetMtime(string, time.Time) error
	SetWriteInPlace(bool)
	Size(string) (int64, error)
	WriteToFile(string, io.Reader) error
}

// Check if fullpath looks like a gdrive path (starting with g: or gdrive:). If
// so, return true and the path without the prefix. Otherwise, return false and
// the path itself.
//
// Returns
//   bool
//   realpath
func isGdrivePath(fullpath string) (bool, string) {
	if strings.HasPrefix(fullpath, "g:") || strings.HasPrefix(fullpath, "gdrive:") {
		idx := strings.Index(fullpath, ":")
		return true, fullpath[idx+1:]
	}
	return false, fullpath
}

// Prints error message and program usage to stderr, exit the program.
func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	}
	fmt.Fprintf(os.Stderr, "Usage%s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var (
		srcvfs   gsyncVfs
		dstvfs   gsyncVfs
		gfs      gsyncVfs
		lfs      gsyncVfs
		srcdir   string
		dstdir   string
		srcpaths []string
	)

	parseFlags()

	// Set verbose level
	log = logger.New("")
	if opt.verbose > 0 {
		log.SetVerboseLevel(int(opt.verbose))
	}

	srcpaths, dstdir, err := getSourceDest()
	if err != nil {
		usage(err)
	}

	// Initialize virtual filesystems
	gfs, err = initGdriveVfs(opt.clientId, opt.clientSecret, opt.code)
	if err != nil {
		log.Fatal(err)
	}
	lfs = localvfs.NewLocalFileSystem()
	dstvfs = lfs
	isDstGdrive, dstPath := isGdrivePath(dstdir)
	if isDstGdrive {
		dstvfs = gfs
	}
	if opt.inplace {
		dstvfs.SetWriteInPlace(true)
	}

	// Treat each path separately
	for _, srcdir = range srcpaths {
		isSrcGdrive, srcPath := isGdrivePath(srcdir)

		// Select VFSes according to path type
		srcvfs = lfs
		if isSrcGdrive {
			srcvfs = gfs
		}

		// Sync
		err = sync(srcPath, dstPath, srcvfs, dstvfs)
		if err != nil {
			log.Fatal(err)
		}
	}
}
