package main

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

const (
	// Flag defaults
	DEFAULT_OPT_VERBOSE = false
	DEFAULT_OPT_DRY_RUN = false
)

type cmdLineOpts struct {
	clientId     string
	clientSecret string
	code         string
	dryrun       bool
	exclude      string
	inplace      bool
	verbose      bool
}

var (
	// Command line Flags
	opt cmdLineOpts

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

// Retrieve the sources and destination from the command-line, performing basic sanity checking.
//
// Returns:
// 	[]string: source paths
// 	string: destination directory
// 	error
func getSourceDest() ([]string, string, error) {
	var srcpaths []string

	if flag.NArg() < 2 {
		return nil, "", fmt.Errorf("Must specify source and destination directories")
	}

	// All arguments but last are considered to be sources
	for ix := 0; ix < flag.NArg()-1; ix++ {
		srcpaths = append(srcpaths, flag.Arg(ix))
	}
	dst := flag.Arg(flag.NArg() - 1)

	return srcpaths, dst, nil
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

	// Parse command line
	flag.StringVar(&opt.clientId, "id", "", "Client ID")
	flag.StringVar(&opt.clientSecret, "secret", "", "Client Secret")
	flag.StringVar(&opt.code, "code", "", "Authorization Code")
	flag.BoolVar(&opt.dryrun, "dry-run", DEFAULT_OPT_DRY_RUN, "Dry-run mode")
	flag.BoolVar(&opt.dryrun, "n", DEFAULT_OPT_DRY_RUN, "Dry-run mode (shorthand)")
	flag.BoolVar(&opt.verbose, "verbose", DEFAULT_OPT_VERBOSE, "Verbose Mode")
	flag.BoolVar(&opt.verbose, "v", DEFAULT_OPT_VERBOSE, "Verbose mode (shorthand)")
	flag.BoolVar(&opt.inplace, "inplace", false, "Upload files in place (faster, but may leave incomplete files behind if program dies)")
	flag.Parse()

	// Set verbose level
	log = logger.New("")
	if opt.verbose {
		log.SetVerboseLevel(1)
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
