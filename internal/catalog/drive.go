
package catalog

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"
	"path"
	
        "gopkg.in/yaml.v3"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_FILE = "config.yml"
const CONFIG_CATALOG = "catalog"

type config struct {
	Type string
	Location string
	Description string
}

type Drive struct{
	Name string
	Description string
	Catalog string
	//catalogPath string
	//catalog *Catalog
	Storage storage.Storage
}

// Store catalogs locally.

func FetchCatalog(dr Drive) (Catalog, error) {
	content, err := ioutil.ReadFile(dr.Catalog)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %w", err)
	}
	cat, err := NewCatalog(content)
	return cat, nil
}

func UpdateCatalog(dr Drive, cat Catalog) error {
	flatCat := Flatten(cat)
	catFile := []byte(strings.Join(flatCat, "\n") + "\n")
	// Have we created a .tmp backup backup?
	made_tmp := false
	// Backup catalog.bak into catalog.tmp if it exists.
	if _, err := os.Stat(dr.Catalog + ".bak"); err == nil {
		// Backup exists, so keep it.
		if err := os.Rename(dr.Catalog + ".bak", dr.Catalog + ".tmp"); err != nil {
			return fmt.Errorf("cannot temporarily preserve backup catalog")
		}
		made_tmp = true
	}
	// Backup catalog into catalog.bak.
	if err := os.Rename(dr.Catalog, dr.Catalog + ".bak"); err != nil {
		if made_tmp { 
			if err2 := os.Rename(dr.Catalog + ".tmp", dr.Catalog + ".bak"); err2 != nil {
				return fmt.Errorf("cannot create backup catalog (%w) or restore tmp backup (%w)", err, err2)
			}
		}
		return fmt.Errorf("cannot create backup catalog: %w", err)
	}
	// Write catalog.
	err := ioutil.WriteFile(dr.Catalog, catFile, 0600)
	if err != nil {
		if err2 := os.Rename(dr.Catalog + ".bak", dr.Catalog); err2 != nil {
			return fmt.Errorf("cannot update catalog (%w) or restore backup (%w)", err, err2)
		}
		if made_tmp {
			if err2 := os.Rename(dr.Catalog + ".tmp", dr.Catalog + ".bak"); err2 != nil {
				return fmt.Errorf("cannot update catalog (%w) or restore tmp backup (%w)", err, err2)
			}
		}
		return fmt.Errorf("cannot update catalog: %s", err)
	}
	// Remove catalog.tmp since no longer needed.
	if made_tmp {
		if err := os.Remove(dr.Catalog + ".tmp"); err != nil {
			return fmt.Errorf("cannot remote tmp backup: %w", err)
		}
	}
	return nil
}

func ReadDrives() (map[string]Drive, error) {
	home, err := os.UserHomeDir()
	if err != nil { 
		return nil, fmt.Errorf("cannot get home directory: %v", err)
	}
	configFolder := path.Join(home, CONFIG_FOLDER)
	info, err := os.Stat(configFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(configFolder, 0700)
		if err != nil {
			return nil, fmt.Errorf("cannot create %s directory: %w", configFolder, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot access %s directory: %w", configFolder, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path %s not a directory", configFolder)
	}
	files, err := ioutil.ReadDir(configFolder)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s directory: %w", configFolder, err)
	}
	drives:= make(map[string]Drive)
	for _, f := range files {
		if f.IsDir() {
			yamlFile, err := ioutil.ReadFile(path.Join(configFolder, f.Name(), CONFIG_FILE))
			// Skip errors silently.
			if err == nil {
				config := config{}
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
					drives[f.Name()] = Drive{f.Name(), config.Description, catalogPath, store}
				}
			}
		}
	}
	return drives, nil
}
