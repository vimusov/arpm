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
	"archive/tar"
	"bufio"
	"fmt"
	"github.com/DataDog/zstd"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	pkgExt        = ".pkg.tar.zst"
	pkgWildcard   = "*" + pkgExt
	fileChunkSize = 16 * 1048576
)

func getPkgName(path string) (string, error) {
	pkgFile, openErr := os.Open(path)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := pkgFile.Close(); closeErr != nil {
			logError(closeErr, "Unable to close pkg file '%s'", pkgFile)
		}
	}()

	reader := zstd.NewReader(bufio.NewReaderSize(pkgFile, fileChunkSize))
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			logError(closeErr, "Unable to close gzip reader")
		}
	}()

	dbTar := tar.NewReader(reader)
	content := make([]byte, 65536)

	parseName := func(desc string) string {
		for _, line := range strings.Split(desc, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == "pkgname" {
				return strings.TrimSpace(parts[1])
			}
		}
		return ""
	}

	for {
		header, tarErr := dbTar.Next()
		if tarErr != nil {
			if tarErr == io.EOF {
				break
			}
			return "", tarErr
		}
		if header == nil {
			return "", fmt.Errorf("invalid TAR header in '%s'", path)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != ".PKGINFO" {
			continue
		}
		readSize, readErr := dbTar.Read(content)
		if readErr != nil && readErr != io.EOF {
			return "", fmt.Errorf("info file in '%s' is too big", path)
		}
		if readSize == 0 {
			return "", fmt.Errorf("zero bytes read from 'info' file in '%s'", path)
		}
		if name := parseName(string(content)); name != "" {
			return name, nil
		}
		break
	}
	return "", fmt.Errorf("no pkgname in '%s'", path)
}

func loadPkgNames(dirPath string) (map[string][]string, error) {
	paths, globErr := filepath.Glob(filepath.Join(dirPath, pkgWildcard))
	if globErr != nil {
		return nil, globErr
	}
	result := make(map[string][]string)
	for _, path := range paths {
		name, nameErr := getPkgName(path)
		if nameErr != nil {
			return nil, nameErr
		}
		_, found := result[name]
		if found {
			result[name] = append(result[name], path)
		} else {
			result[name] = []string{path}
		}
	}
	return result, nil
}
