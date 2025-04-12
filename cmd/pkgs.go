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
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func lsPkgsHandler(rootDir string, c echo.Context) error {
	branch := c.Param("branch")
	if branch == "" {
		return c.NoContent(http.StatusNotFound)
	}
	if name := c.QueryParam("name"); name != "" {
		return c.File(filepath.Join(rootDir, branch, name))
	}
	paths, globErr := filepath.Glob(filepath.Join(rootDir, branch, pkgWildcard))
	if globErr != nil {
		logError(globErr, "Unable to glob pkg in '%s/%s'", rootDir, branch)
		return c.NoContent(http.StatusInternalServerError)
	}
	var names []string
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}
	if len(names) == 0 {
		return c.String(http.StatusOK, "No entries.")
	}
	sort.Strings(names)
	return c.String(http.StatusOK, strings.Join(names, "\n"))
}

func addPkgHandler(rootDir string, c echo.Context) error {
	branch := c.Param("branch")
	if branch == "" {
		return c.NoContent(http.StatusNotFound)
	}
	name := c.QueryParam("name")
	if name == "" {
		return c.NoContent(http.StatusBadRequest)
	}
	branchDir := filepath.Join(rootDir, branch)
	pkgs, pkgsErr := loadPkgNames(branchDir)
	if pkgsErr != nil {
		logError(pkgsErr, "Unable to load pkg names from '%s'", branchDir)
		return c.NoContent(http.StatusInternalServerError)
	}
	tmpPath := filepath.Join(branchDir, "tmp_"+name+"_pmt")
	logInfo("Storing '%s'", tmpPath)
	if saveErr := saveFile(tmpPath, c.Request().Body); saveErr != nil {
		logError(saveErr, "Unable to save pkg to '%s'", tmpPath)
		return c.NoContent(http.StatusInternalServerError)
	}
	newName, pkgErr := getPkgName(tmpPath)
	if pkgErr != nil {
		logError(pkgErr, "Unable to load pkg name from '%s'", tmpPath)
		defer rmFile(tmpPath)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, path := range pkgs[newName] {
		rmFile(path)
	}
	newPath := filepath.Join(branchDir, name)
	logInfo("Moving '%s'=>'%s'", tmpPath, newPath)
	if renameErr := os.Rename(tmpPath, newPath); renameErr != nil {
		logError(renameErr, "Unable to rename pkg from '%s' to '%s'", tmpPath, newPath)
		defer rmFile(tmpPath)
		defer rmFile(newPath)
		return c.NoContent(http.StatusInternalServerError)
	}
	if rebuildErr := rebuildDatabase(branchDir, branch); rebuildErr != nil {
		defer rmFile(newPath)
		logError(rebuildErr, "Unable to rebuild database")
		return c.NoContent(http.StatusInternalServerError)
	}
	logInfo("Added '%s'", newPath)
	return c.NoContent(http.StatusCreated)
}

func rmPkgHandler(rootDir string, c echo.Context) error {
	branch := c.Param("branch")
	if branch == "" {
		return c.NoContent(http.StatusNotFound)
	}
	names := c.QueryParam("name")
	if names == "" {
		return c.NoContent(http.StatusBadRequest)
	}
	branchDir := filepath.Join(rootDir, branch)
	pkgs, pkgsErr := loadPkgNames(branchDir)
	if pkgsErr != nil {
		logError(pkgsErr, "Unable to load pkg names from '%s'", branchDir)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, name := range strings.Split(names, ",") {
		paths := pkgs[name]
		if strings.HasSuffix(name, pkgExt) {
			paths = append(paths, filepath.Join(branchDir, name))
		}
		for _, path := range paths {
			rmFile(path)
		}
	}
	if rebuildErr := rebuildDatabase(branchDir, branch); rebuildErr != nil {
		logError(rebuildErr, "Unable to rebuild database")
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}
