
package main

func initializeDrives() (map[string]drive, drive) {
	drivesList := [...]drive{
		drive{
			"test",
			"gcs",
			"vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002",
			"7b5d41cc-86d6-11ec-a8a3-0242ac120002",
		},
	}
	drives := make(map[string]drive)
	for _, dr := range drivesList {
		drives[dr.name] = dr
	}
	default_drive := drives["test"]
	return drives, default_drive
}
