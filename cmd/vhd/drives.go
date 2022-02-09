
package main

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"
	"path"
	
        "gopkg.in/yaml.v3"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
	"rpucella.net/virtual-hard-drive/internal/catalog"
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_FILE = "config.yml"
const CONFIG_CATALOG = "catalog"

type Config struct {
	Type string
	Location string
}

type drive struct{
	name string
	catalog string
	storage storage.Storage
}

// Store catalogs locally.

func fetchCatalog(dr drive) (catalog.Catalog, error) {
	content, err := ioutil.ReadFile(dr.catalog)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %w", err)
	}
	cat, err := catalog.NewCatalog(content)
	return cat, nil
}

func updateCatalog(dr drive, cat catalog.Catalog) error {
	flatCat := catalog.Flatten(cat)
	catFile := []byte(strings.Join(flatCat, "\n") + "\n")
	// TODO: Backup old catalog.
	err := ioutil.WriteFile(dr.catalog, catFile, 0600)
	if err != nil {
		return fmt.Errorf("cannot update catalog: %s", err)
	}
	return nil
}

func readDrives() (map[string]drive, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return nil, fmt.Errorf("cannot read HOME environment variable")
	}
	configFolder := path.Join(home, CONFIG_FOLDER)
	info, err := os.Stat(configFolder)
	if os.IsNotExist(err) {
		fmt.Println("creating config folder")
		err := os.Mkdir(configFolder, 0700)
		if err != nil {
			return nil, fmt.Errorf("cannot create %s directory: %w", configFolder, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot access %s directory: %w", configFolder, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path %s not a directory", configFolder)
	}
	fmt.Println("reading config folder")
	files, err := ioutil.ReadDir(configFolder)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s directory: %w", configFolder, err)
	}
	drives:= make(map[string]drive)
	for _, f := range files {
		if f.IsDir() {
			fmt.Println("found", f.Name())
			yamlFile, err := ioutil.ReadFile(path.Join(configFolder, f.Name(), CONFIG_FILE))
			// Skip errors silently.
			if err == nil {
				config := Config{}
				err := yaml.Unmarshal(yamlFile, &config)
				// Again, skip errors silently.
				if err == nil {
					var store storage.Storage
					if config.Type == "gcs" {
						store = storage.NewGoogleCloud(config.Location)
					} else if config.Type == "local" {
						store = storage.NewLocalFileSystem(config.Location)
					} else {
						// Unknown type - skip silently.
						continue
					}
					catalogPath := path.Join(configFolder, f.Name(), CONFIG_CATALOG)
					drives[f.Name()] = drive{f.Name(), catalogPath, store}
				}
			}
		}
	}
	return drives, nil
}
