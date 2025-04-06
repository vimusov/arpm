package main

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
