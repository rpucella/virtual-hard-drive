
package catalog

import (
	"fmt"
	"os"
	"path"
	"time"
	"database/sql"
	
	_ "github.com/mattn/go-sqlite3"
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_SQLITE = "catalog.db"

type config struct {
	Type string
	Location string
	Description string
}

type sqlCatalog struct {
	dbPath string
}

func openDB(c *sqlCatalog) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", c.dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open db file: %w", err)
	}
	return db, nil
}

func (c *sqlCatalog) FetchDrives() (map[int]DriveDescriptor, error) {
	db, err := openDB(c)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	
	drives:= make(map[int]DriveDescriptor)
	rows, err := db.Query("SELECT * FROM drives")
	var id int
	var name string
	var description string
	var host string
	var address string
	for rows.Next() {
		err = rows.Scan(&id, &name, &description, &host, &address)
		if err != nil {
			return nil, fmt.Errorf("error reading drives table: %w", err)
		}
		drives[id] = DriveDescriptor{id, name, host, address, description}
	}
	// db.Close()
	return drives, nil
}

func (c *sqlCatalog) FetchDirectories(driveId int) (map[int]DirectoryDescriptor, error) {
	db, err := openDB(c)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	
	rows, err := db.Query("SELECT id, name, parentId FROM directories WHERE driveId = ?", driveId)
	if err != nil {
		return nil, fmt.Errorf("db.Query(directories): %w", err)
	}
	directories := make(map[int]DirectoryDescriptor)
	var id int
	var name string
	var parentId int
	for rows.Next() {
		err = rows.Scan(&id, &name, &parentId)
		if err != nil {
			return nil, fmt.Errorf("error reading directories table: %w", err)
		}
		directories[id] = DirectoryDescriptor{id, name, parentId}
	}
	return directories, nil
}

func (c *sqlCatalog) FetchFiles(driveId int) (map[int]FileDescriptor, error) {
	db, err := openDB(c)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	
	rows, err := db.Query("SELECT id, name, directoryId, uuid, created, updated, metadata FROM files WHERE driveId = ?", driveId)
	if err != nil {
		return nil, fmt.Errorf("db.Query(files): %w", err)
	}
	files := make(map[int]FileDescriptor)
	var id int
	var name string
	var directoryId int
	var uuid string
	var created int64
	var updated int64
	var metadata string
	for rows.Next() {
		err = rows.Scan(&id, &name, &directoryId, &uuid, &created, &updated, &metadata)
		if err != nil {
			return nil, fmt.Errorf("error reading files table: %w", err)
		}
		upTime := time.Unix(updated, 0)
		crTime := time.Unix(created, 0)
		files[id] = FileDescriptor{id, name, directoryId, uuid, crTime, upTime, metadata}
	}
	return files, nil
}


func (c *sqlCatalog) CreateFile(driveId int, name string, uuid string, dirId int, created time.Time, updated time.Time, metadata string) (int, error) {
	db, err := openDB(c)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	if _, err := db.Exec("INSERT INTO files (driveId, name, directoryId, uuid, created, updated, metadata) values (?, ?, ?, ?, ?, ?, ?)", driveId, name, dirId, uuid, created.Unix(), updated.Unix(), metadata); err != nil {
		return 0, fmt.Errorf("db.Exec: %w", err)
	}
	row := db.QueryRow("SELECT last_insert_rowid()")
	var id int64
	if err := row.Scan(&id); err != nil { 
		return 0, fmt.Errorf("db.QueryRow: %w", err)
	}
	db.Close()
	return int(id), nil
}

func (c *sqlCatalog) CreateDirectory(driveId int, name string, parentId int) (int, error) {
	db, err := openDB(c)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	if _, err := db.Exec("INSERT INTO directories (driveId, name, parentId) values (?, ?, ?)", driveId, name, parentId); err != nil {
		return 0, fmt.Errorf("db.Exec: %w", err)
	}
	row := db.QueryRow("SELECT last_insert_rowid()")
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("db.QueryRow: %w", err)
	}
	db.Close()
	return int(id), nil
}

func (c *sqlCatalog) UpdateFile(id int, name string, dirId int) error {
	db, err := openDB(c)
	if err != nil {
		return err
	}
	defer db.Close()
	
	if _, err := db.Exec("UPDATE files SET name = ?, directoryId = ? where id = ?", name, dirId, id); err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	db.Close()
	return nil
}

func (c *sqlCatalog) UpdateDirectory(id int, name string, parentId int) error {
	db, err := openDB(c)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec("UPDATE directories SET name = ?, parentId = ? where id = ?", name, parentId, id); err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	db.Close()
	return nil
}

func Load() (Catalog, error) {
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
	result := &sqlCatalog{
		path.Join(configFolder, CONFIG_SQLITE),
	}
	return result, nil
}

func (c *sqlCatalog) CountFilesInDirectory(dirId int) (int, error) {
	db, err := openDB(c)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	
	row := db.QueryRow(`  with recursive subfolders(name, id) as (
                                     select name, id from directories where parentId = ?
                                     union all
                                     select directories.name, directories.id 
                                       from directories, subfolders 
                                       where directories.parentId = subfolders.id
                                   ) select count(*) from files where directoryId = ? or directoryId in (select id from subfolders)`, dirId, dirId)
	var count int
	if err := row.Scan(&count); err != nil { 
		return 0, fmt.Errorf("db.QueryRow: %w", err)
	}
	db.Close()
	return count, nil
}

func (c *sqlCatalog) CountFilesInDrive(driveId int) (int, error) {
	db, err := openDB(c)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	
	row := db.QueryRow(`select count(*) from files where driveId = ?`, driveId)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("db.QueryRow: %w", err)
	}
	db.Close()
	return count, nil
}
