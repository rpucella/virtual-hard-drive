
package util

import (
	"os"
	"path"
	"fmt"
)

const CONFIG_FOLDER = ".vhd"

func ConfigFolder() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil { 
		return "", fmt.Errorf("cannot get home directory: %w", err)
	}
	configFolder := path.Join(home, CONFIG_FOLDER)
	info, err := os.Stat(configFolder)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("config folder %s does not exist", configFolder)
	} else if err != nil {
		return "", fmt.Errorf("cannot access %s directory: %w", configFolder, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("path %s not a directory", configFolder)
	}
	return configFolder, nil
}

func ConfigFile(name string) (string, error) {
	configFolder, err := ConfigFolder()
	if err != nil {
		return "", err
	}
	return path.Join(configFolder, name), nil
}

