package main

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
		return true, nil
	}

	return false, nil
}

// Return true if the passed path matches one of the patterns in the exclusion
// list (opt.exclude).
//
// Return:
//   bool
//   error

func excluded(pathname string) (bool, error) {
	for _, excpat := range opt.exclude {
		match, err := filepath.Match(excpat, pathname)
		if err != nil {
			return false, err
		}
		if match {
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

		// If the source path ends in a slash, we'll copy the *contents* of the
		// source directory to the destination. If it doesn't, we'll create a
		// directory inside the destination. This matches rsync's behavior
		//
		// Ex:
		// /a/b/c/ -> foo = /foo/<files>...
		// /a/b/c  -> foo = /foo/c/<files>...

		// Default == copy files INTO directory at destination
		dst := path.Join(dstdir, src[len(srcpath):])

		// If source does not end in "/", we create the directory specified
		// by srcpath as the first level inside the destination.
		if !strings.HasSuffix(srcpath, "/") {
			sdir := strings.Split(srcpath, "/")
			if len(sdir) > 1 {
				last := len(sdir) - 1
				ssrc := strings.Split(src, "/")
				dst = path.Join(dstdir, strings.Join(ssrc[last:], "/"))
			}
		}

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
