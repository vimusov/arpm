package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func rmFile(path string) {
	logInfo("Removing '%s'", path)
	if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
		logError(rmErr, "Unable to remove '%s'", path)
	}
}

func saveFile(path string, reader io.ReadCloser) error {
	fp, fpErr := os.Create(path)
	if fpErr != nil {
		return fpErr
	}
	defer func() {
		if closeErr := fp.Close(); closeErr != nil {
			logError(closeErr, "Failed to close file")
		}
		if readerErr := reader.Close(); readerErr != nil {
			logError(readerErr, "Failed to close reader")
		}
	}()
	if _, copyErr := io.Copy(fp, reader); copyErr != nil {
		return copyErr
	}
	return fp.Sync()
}

func rebuildDatabase(dirPath, branch string) error {
	dbPaths, dbGlobErr := filepath.Glob(filepath.Join(dirPath, fmt.Sprintf("%s.*", branch)))
	if dbGlobErr != nil {
		return dbGlobErr
	}
	pkgPaths, pkgGlobErr := filepath.Glob(filepath.Join(dirPath, pkgWildcard))
	if pkgGlobErr != nil {
		return pkgGlobErr
	}
	args := []string{filepath.Join(dirPath, fmt.Sprintf("%s.db.tar.gz", branch))}
	args = append(args, pkgPaths...)
	for _, path := range dbPaths {
		rmFile(path)
	}
	if len(pkgPaths) == 0 {
		return nil
	}
	sargs := strings.Join(args, " ")
	cmd := exec.Command("repo-add", args...)
	stdout, execErr := cmd.CombinedOutput()
	lines := strings.ReplaceAll(string(stdout), "\n", "\\n")
	if execErr != nil {
		logError(execErr, "Failed to execute repo-add '%s', stdout='%s'", sargs, lines)
	} else {
		logDebug("exec repo-add '%s': '%s'", sargs, lines)
	}
	return execErr
}
