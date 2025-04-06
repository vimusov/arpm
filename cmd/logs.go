package main

import (
	"fmt"
	"os"
)

var debugMode bool

func logDebug(format string, args ...any) {
	if debugMode {
		_, _ = fmt.Fprintf(os.Stderr, "DEBUG: "+format+".\n", args...)
	}
}

func logInfo(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "INFO: "+format+".\n", args...)
}

func logError(err error, format string, args ...any) {
	msg := fmt.Sprintf("ERROR: "+format, args...)
	if err == nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s.\n", msg)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %s.\n", msg, err)
	}
}
