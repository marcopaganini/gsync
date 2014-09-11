package logger

// Simple customized logger for gsync.
//
// (C) 2014 by Marco Paganini <paganini AT paganini DOT net>

import (
	"log"
	"os"
)

// Base Logger struct using anonymous *log.Logger method
type Logger struct {
	*log.Logger
	verbose int
	debug   int
}

// Creates a new Logger instance based on log and returns it
//
// Returns:
//   logger instance
func New(prefix string) *Logger {
	o := log.New(os.Stderr, prefix, 0)
	return &Logger{o, 0, 0}
}

// Set Verbose level
func (o *Logger) SetVerboseLevel(n int) {
	o.verbose = n
}

// Set Debugging level
func (o *Logger) SetDebugLevel(n int) {
	o.debug = n
}

// PrintLn the message if the current verbose level >= 'level'
func (o *Logger) Verboseln(level int, v ...interface{}) {
	if o.verbose >= level {
		o.Println(v)
	}
}

// Printf the message and parameters if the current verbose level >= 'level'
func (o *Logger) VerboseF(level int, format string, v ...interface{}) {
	if o.verbose >= level {
		o.Printf(format, v)
	}
}

// PrintLn the message if the current Debug level >= 'level'
func (o *Logger) Debugln(level int, v ...interface{}) {
	o.SetFlags(log.Lshortfile)
	if o.debug >= level {
		o.Println(v)
	}
	o.SetFlags(0)
}

// Printf the message and parameters if the current Debug level >= 'level'
func (o *Logger) DebugF(level int, format string, v ...interface{}) {
	o.SetFlags(log.Lshortfile)
	if o.debug >= level {
		o.Printf(format, v)
	}
	o.SetFlags(0)
}
