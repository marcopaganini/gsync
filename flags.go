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

type multiString []string

type cmdLineOpts struct {
	clientId     string
	clientSecret string
	code         string
	dryrun       bool
	exclude      multiString
	inplace      bool
	verbose      bool
}

var (
	// Command line Flags
	opt cmdLineOpts
)

// Definitions for the custom flag type multiString

// Return the string representation of the flag.
// The String method's output will be used in diagnostics.
func (m *multiString) String() string {
	return fmt.Sprint(*m)
}

// Append 'value' to multistring. This allows options like --optx val1 --optx
// val2, etc. multiString will be set to an array containing all the options.
// 'value' is split by commas so we have to split it.
func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
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
	flag.Var(&opt.exclude, "exclude", "List of paths to exclude (glob)")
	flag.Parse()
}
