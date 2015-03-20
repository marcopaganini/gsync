package main

// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Directory pairs for sync post-processing of directories
type dirpair struct {
	src string
	dst string
}

// Generate a destination path based on the source directory and
// path under that directory.
func destPath(srcdir string, dstdir string, srcfile string) string {
	var (
		sdir     []string
		sfile    []string
		ddir     []string
		dst      []string
		barefile []string
	)

	// Start with source dir with all relative path elements removed.
	sdir = []string{}
	for _, v := range strings.Split(srcdir, "/") {
		if v != "." && v != ".." && v != "" {
			sdir = append(sdir, v)
		}
	}

	// Convert to string removing empty elements, etc
	sfile = []string{}
	for _, v := range strings.Split(srcfile, "/") {
		if v != "." && v != ".." && v != "" {
			sfile = append(sfile, v)
		}
	}
	ddir = []string{}
	for _, v := range strings.Split(dstdir, "/") {
		if v != "." && v != ".." && v != "" {
			ddir = append(ddir, v)
		}
	}

	// source file with the source directory part removed
	barefile = sfile[len(sdir):]

	if strings.HasSuffix(srcdir, "/") {
		// Copy files INTO directory at destination.  full destination path is
		// the destionation directory + the source file with srcdir removed.
		dst = ddir
		dst = append(dst, barefile...)
	} else {
		// Original path does not end in "/". We preserve the
		// last path element of srcdir into the destination.
		dst = ddir
		if len(sdir) > 0 {
			dst = append(dst, sdir[len(sdir)-1])
		}
		dst = append(dst, barefile...)
	}

	return strings.Join(dst, "/")
}

// Determine if we need to copy the file pointed by srcpath in srcvfs to
// the file dstpath in dstvfs.
//
// Return:
// 	 bool
// 	 error
func needToCopy(srcvfs gsyncVfs, dstvfs gsyncVfs, srcpath string, dstpath string) (bool, error) {
	// If destination doesn't exist we need to copy
	exists, err := dstvfs.FileExists(dstpath)
	if err != nil {
		return false, err
	}
	if !exists {
		log.Verbosef(2, "needToCopy: destination file %q does not exist; will copy.", srcpath)
		return true, nil
	}

	// If destination exists, we check mtimes truncated to the nearest second
	srcMtime, err := srcvfs.Mtime(srcpath)
	if err != nil {
		return false, err
	}
	dstMtime, err := dstvfs.Mtime(dstpath)
	if err != nil {
		return false, err
	}

	srcMtime = srcMtime.Truncate(time.Second)
	dstMtime = dstMtime.Truncate(time.Second)

	if srcMtime.After(dstMtime) {
		log.Verbosef(2, "needToCopy: %q: source is newer destination (%v > %v); will copy.", srcpath, srcMtime, dstMtime)
		return true, nil
	}

	log.Verbosef(2, "needToCopy: %q: source is older than destination (%v <= %v); will not copy.", srcpath, srcMtime, dstMtime)
	return false, nil
}

// Return true if the passed path matches one of the patterns in the exclusion
// list (opt.exclude).
//
// Return:
//   bool
//   error

func excluded(pathname string) (bool, error) {
	fname := path.Base(pathname)
	for _, excpat := range opt.exclude {
		log.Verbosef(3, "attempting to match %q to pattern %q", pathname, excpat)
		match, err := filepath.Match(excpat, fname)
		if err != nil {
			return false, err
		}
		if match {
			log.Verbosef(3, "excluding %q: matched %q", pathname, excpat)
			return match, err
		}
	}
	return false, nil
}

// Copy the content of all files/directories pointed by srcpath into dstdir.
// If srcpath is a file, the file will be copied. If it is a directory, the
// entire subtree will be copied.  Dstdir must be a directory.
//
// Like rsync, a source path ending in slash means "copy the contents of this
// directory into the destination" whereas a path not ending in a slash means
// "copy this directory and its contents into the destination."
//
// Files/directories are only copied if needed (based on the modification date
// of the file on both filesystems.) This function uses the srcvfs and dstvfs
// VFS objects to perform operations on the respective filesystems.
//
// Return:
// 	 error
func sync(srcpath string, dstdir string, srcvfs gsyncVfs, dstvfs gsyncVfs) error {
	var (
		srctree  []string
		dirpairs []dirpair
	)

	// Destination must exist and be a directory
	exists, err := dstvfs.FileExists(dstdir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Destination \"%s\" does not exist", dstdir)
	}

	isdir, err := dstvfs.IsDir(dstdir)
	if err != nil {
		return err
	}
	if !isdir {
		return fmt.Errorf("Destination \"%s\" is not a directory/folder", dstdir)
	}

	// Special case: If the source path is not a directory, we short circuit
	// the FileTree method here and set srctree to that single file.
	isdir, err = srcvfs.IsDir(srcpath)
	if err != nil {
		return err
	}
	if isdir {
		srctree, err = srcvfs.FileTree(srcpath)
		if err != nil {
			return err
		}
	} else {
		srctree = []string{srcpath}
	}

	// Guarantee that we'll process a directory before files inside it
	sort.Strings(srctree)

	for _, src := range srctree {
		// Check for exclusions (--exclude)
		exc, err := excluded(src)
		if err != nil {
			return err
		}
		if exc {
			log.Verboseln(2, src, "excluded from copy")
			continue
		}

		dst := destPath(srcpath, dstdir, src)

		isdir, err := srcvfs.IsDir(src)
		if err != nil {
			return err
		}
		isregular, err := srcvfs.IsRegular(src)
		if err != nil {
			return err
		}

		// Start sync operation

		if isdir {
			// Create destination dir if needed
			exists, err := dstvfs.FileExists(dst)
			if err != nil {
				return err
			}
			if !exists {
				log.Verboseln(1, dst)
				if !opt.dryrun {
					err := dstvfs.Mkdir(dst)
					if err != nil {
						return err
					}
				}
			}
			// Save directory for post processing
			d := dirpair{src, dst}
			dirpairs = append(dirpairs, d)
		} else if isregular {
			copyNeeded, err := needToCopy(srcvfs, dstvfs, src, dst)
			if err != nil {
				return err
			}

			if copyNeeded {
				if !opt.dryrun {
					r, err := srcvfs.ReadFromFile(src)
					if err != nil {
						log.Printf("Warning: Skipping \"%s\": %v\n", src, err)
						continue
					}
					err = dstvfs.WriteToFile(dst, r)
					if err != nil {
						return err
					}
					// Set destination mtime == source mtime
					mtime, err := srcvfs.Mtime(src)
					if err != nil {
						return err
					}
					err = dstvfs.SetMtime(dst, mtime)
					if err != nil {
						return err
					}
				}
				log.Verboseln(1, dst)
			}
		} else {
			log.Printf("Warning: Skipping \"%s\": not a regular file or directory.\n", src)
			continue
		}
	}

	// Set the mtimes of all destination directories to the original mtimes.
	// We have to do it here (and bottom first!) because in certain filesystems,
	// updating files inside directories will also change the directory mtime.

	if !opt.dryrun {
		for ix := len(dirpairs) - 1; ix >= 0; ix-- {
			src := dirpairs[ix].src
			dst := dirpairs[ix].dst

			mtime, err := srcvfs.Mtime(src)
			if err != nil {
				return err
			}
			err = dstvfs.SetMtime(dst, mtime)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
