
package main

import (
	"fmt"
	"strings"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
	"rpucella.net/virtual-hard-drive/internal/catalog"
)

type drive_info struct {
	name string
	provider string
	bucket string
	// encryption string
}

func initializeDrives() (map[string]drive, drive) {
	drivesList := [...]drive_info{
		drive_info{
			"gcs-test",
			"gcs",
			"vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002",
		},
		drive_info{
			"local-test",
			"local",
			"/Users/riccardo/git/virtual-hard-drive/local_test",
		},
	}
	drives := make(map[string]drive)
	for _, dr := range drivesList {
		var store storage.Storage
		if dr.provider == "gcs" {
			store = storage.NewGoogleCloud(dr.bucket)
		} else if dr.provider == "local" {
			store = storage.NewLocalFileSystem(dr.bucket)
		} else {
			// Unrecognized provider - skip.
			continue
		}
		drives[dr.name] = drive{
			dr.name,
			store,
		}
	}
	default_drive := drives["local-test"]
	return drives, default_drive
}

func fetchCatalog(dr drive) (catalog.Catalog, error) {
	path, err := dr.storage.CatalogToPath("catalog")
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %w", err)
	}
	content, err := dr.storage.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %s", err)
	}
	cat, err := catalog.NewCatalog(content)
	return cat, nil
}

func updateCatalog(dr drive, cat catalog.Catalog) error {
	path, err := dr.storage.CatalogToPath("catalog")
	if err != nil {
		return fmt.Errorf("cannot update catalog: %w", err)
	}
	flatCat := catalog.Flatten(cat)
	catFile := []byte(strings.Join(flatCat, "\n") + "\n")
	// TODO: Backup old catalog.
	err = dr.storage.WriteFile(path, catFile)
	if err != nil {
		return fmt.Errorf("cannot update catalog: %s", err)
	}
	return nil
}
