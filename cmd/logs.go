package main

/*
   This file is part of arpm.

   arpm is free software: you can redistribute it and/or modify it under the terms
   of the GNU General Public License as published by the Free Software Foundation, either
   version 3 of the License, or (at your option) any later version.

   arpm is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
   without even the implied warranty     of MERCHANTABILITY or FITNESS FOR A PARTICULAR
   PURPOSE. See the GNU General Public License for more details.

   You should have received a copy of the GNU General Public License along with arpm.
   If not, see <https://www.gnu.org/licenses/>.
*/

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
