package main

// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import "testing"

func TestDestPath(t *testing.T) {
	paths := [][]string{
		[]string{"/d1", "/d1/foo", "dest/d1/foo"},
		[]string{"/d1", "/d1/foo/bar", "dest/d1/foo/bar"},
		[]string{"/d1/", "/d1/foo", "dest/foo"},
		[]string{"/d1/", "/d1/foo/bar", "dest/foo/bar"},
		[]string{"../d1", "../d1/foo/bar", "dest/d1/foo/bar"},
		[]string{"../d1/", "../d1/foo/bar", "dest/foo/bar"},
		[]string{".", "./foo", "dest/foo"},
		[]string{".", "./foo/bar", "dest/foo/bar"},
		[]string{"./", "./foo", "dest/foo"},
		[]string{"./", "./foo/bar", "dest/foo/bar"},
		[]string{"../", "../foo", "dest/foo"},
		[]string{"../", "../foo/bar", "dest/foo/bar"},
		[]string{"", "./foo", "dest/foo"},
		[]string{"", "./foo/bar", "dest/foo/bar"},
		[]string{"/", "/foo", "dest/foo"},
		[]string{"/", "/foo/bar", "dest/foo/bar"},
	}

	for _, a := range paths {
		r := destPath(a[0], "/dest", a[1])
		if r != a[2] {
			t.Errorf("srcdir=[%s], srcfile=[%s], Expected \"%s\" got \"%s\"\n", a[0], a[1], a[2], r)
		}
	}
}
