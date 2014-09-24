package main

import (
	"flag"
	"fmt"
)

// This file is part of gsync - A google drive syncer in Go
//
// Command line functions
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

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
)

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

// Parse the command line and set the global opt variable
func parseFlags() {
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
}
