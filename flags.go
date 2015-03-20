package main

// This file is part of gsync, a Google Drive syncer in Go.
// See instructions in the README.md file that accompanies this program.
// (C) 2015 by Marco Paganini <paganini AT paganini DOT net>

import (
	"flag"
	"fmt"
)

const (
	// Flag defaults
	defaultOptVerboseLevel = 0
	defaultOptDryRun       = false
)

type multiString []string
type multiLevelInt int

type cmdLineOpts struct {
	clientID     string
	clientSecret string
	code         string
	dryrun       bool
	exclude      multiString
	inplace      bool
	verbose      multiLevelInt
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

// Definitions for the custom flag type multiLevelInt

// Return the string representation of the flag.
// The String method's output will be used in diagnostics.
func (m *multiLevelInt) String() string {
	return fmt.Sprint(*m)
}

// Increase the value of multiLevelInt. This accepts multiple values
// and sets the variable to the number of times those values appear in
// the command-line. Useful for "verbose" and "Debug" levels.
func (m *multiLevelInt) Set(_ string) error {
	*m++
	return nil
}

// Behave as a bool (i.e. no arguments)
func (m *multiLevelInt) IsBoolFlag() bool {
	return true
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
	flag.StringVar(&opt.clientID, "id", "", "Client ID")
	flag.StringVar(&opt.clientSecret, "secret", "", "Client Secret")
	flag.StringVar(&opt.code, "code", "", "Authorization Code")
	flag.BoolVar(&opt.dryrun, "dry-run", defaultOptDryRun, "Dry-run mode")
	flag.BoolVar(&opt.dryrun, "n", defaultOptDryRun, "Dry-run mode (shorthand)")
	flag.BoolVar(&opt.inplace, "inplace", false, "Upload files in place (faster, but may leave incomplete files behind if program dies)")
	flag.Var(&opt.exclude, "exclude", "List of paths to exclude (glob)")
	flag.Var(&opt.verbose, "verbose", "Verbose mode (use multiple times to increase level)")
	flag.Var(&opt.verbose, "v", "Verbose mode (use multiple times to increase level)")
	flag.Parse()
}
