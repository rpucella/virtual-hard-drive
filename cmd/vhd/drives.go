
package main

import (
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type drive_info struct {
	name string
	provider string
	bucket string
	catalog string
}

func initializeDrives() (map[string]drive, drive) {
	drivesList := [...]drive_info{
		drive_info{
			"test",
			"gcs",
			"vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002",
			"7b5d41cc-86d6-11ec-a8a3-0242ac120002",
		},
	}
	drives := make(map[string]drive)
	for _, dr := range drivesList {
		var store storage.Storage
		if dr.provider == "gcs" {
			store = storage.NewGoogleCloud(dr.bucket)
		}
		drives[dr.name] = drive{
			dr.name,
			dr.catalog,
			store,
		}
	}
	default_drive := drives["test"]
	return drives, default_drive
}
