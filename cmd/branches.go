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
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func addBranchHandler(rootDir string, c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return c.NoContent(http.StatusBadRequest)
	}
	dirPath := filepath.Join(rootDir, name)
	logInfo("Creating branch directory '%s'", dirPath)
	if mkErr := os.MkdirAll(dirPath, 0755); mkErr != nil && !os.IsExist(mkErr) {
		logError(mkErr, "Unable to create directory '%s'", dirPath)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusCreated)
}

func lsBranchesHandler(rootDir string, c echo.Context) error {
	dirs, rootGlobErr := filepath.Glob(filepath.Join(rootDir, "*"))
	if rootGlobErr != nil {
		logError(rootGlobErr, "Unable to glob root directory '%s'", rootDir)
		return c.NoContent(http.StatusInternalServerError)
	}
	var result []string
	for _, branchDir := range dirs {
		paths, brGlobErr := filepath.Glob(filepath.Join(branchDir, pkgWildcard))
		if brGlobErr != nil {
			logError(brGlobErr, "Unable to glob pkg directory '%s'", branchDir)
			return c.NoContent(http.StatusBadRequest)
		}
		result = append(result, fmt.Sprintf("%s: %d item(s)", filepath.Base(branchDir), len(paths)))
	}
	if len(result) == 0 {
		return c.String(http.StatusOK, "No entries.")
	}
	sort.Strings(result)
	return c.String(http.StatusOK, strings.Join(result, "\n"))
}
