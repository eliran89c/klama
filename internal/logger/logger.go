package logger

import (
	"io"
	"log"
)

var (
	// global logger
	logger *log.Logger
)

// Init initializes the logger with the given writer
func Init(w io.Writer) {
	logger = log.New(w, "", log.LstdFlags)
}

// Debug logs debug messages
func Debug(args ...interface{}) {
	if logger == nil {
		Init(io.Discard)
	}

	logger.Println(args...)
}

// Debugf logs formatted debug messages
func Debugf(format string, args ...interface{}) {
	if logger == nil {
		Init(io.Discard)
	}

	logger.Printf(format, args...)
}
