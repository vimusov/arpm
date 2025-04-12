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
	"github.com/BurntSushi/toml"
	"os"
	"strings"
)

const configPath = "~/.config/arpm.toml"

var serverUri string

func loadConfig() error {
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("could not get home directory from '%s': %s", configPath, homeErr)
	}
	var config struct {
		Uri string `toml:"server"`
	}
	_, decodeErr := toml.DecodeFile(strings.Replace(configPath, "~", homeDir, 1), &config)
	if decodeErr != nil {
		return fmt.Errorf("could not parse toml from '%s': %s", configPath, decodeErr)
	}
	serverUri = config.Uri
	return nil
}
