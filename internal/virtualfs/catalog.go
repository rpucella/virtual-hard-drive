
package virtualfs

import (
	"fmt"
	"os"
	"path"
	"time"
	"database/sql"
	
	_ "github.com/mattn/go-sqlite3"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_FILE = "config.yml"
const CONFIG_CATALOG = "catalog"
const CONFIG_SQLITE = "catalog.db"

// Path to ~/.vhd/catalog.db database file.
var pathCatalogDB string

type config struct {
	Type string
	Location string
	Description string
}


func openDB() (*sql.DB, error) {
	if pathCatalogDB == "" {
		// Read drives from .vhd/catalog.db SQLite file.
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
		pathCatalogDB = path.Join(configFolder, CONFIG_SQLITE)
	} 
	db, err := sql.Open("sqlite3", pathCatalogDB)
	if err != nil {
		return nil, fmt.Errorf("cannot open db file: %w", err)
	}
	return db, nil
}

func fetchCatalog(dr *drive) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	directories, err := fetchDirectories(db, dr)
	if err != nil {
		return err
	}
	if err := fetchFiles(db, dr, directories); err != nil {
		return err
	}
	
	db.Close()
	return nil
}

func fetchDirectories(db *sql.DB, dr *drive) (map[int]*vfs_dir, error) {
	directories := make(map[int]*vfs_dir)
	parents := make(map[int]int)
	
	stmt, err := db.Prepare("SELECT id, name, parentId FROM directories WHERE driveId=?")
	if err != nil {
		return nil, fmt.Errorf("db.Prepare(directories): %w", err)
	}
	rows, err := stmt.Query(dr.id)
	if err != nil {
		return nil, fmt.Errorf("stmt.Query(directories): %w", err)
	}
	var id int
	var name string
	var parentId int
	for rows.Next() {
		err = rows.Scan(&id, &name, &parentId)
		if err != nil {
			return nil, fmt.Errorf("error reading directories table: %w", err)
		}
		directories[id] = &vfs_dir{name, make(map[string]VirtualFS), nil, id}
		parents[id] = parentId
	}
	root := dr.root
	dr.top = &vfs_dir{"", make(map[string]VirtualFS), root, -1}
	for id, dir := range directories {
		name := dir.name
		var parent VirtualFS
		if parents[id] < 0 {
			parent = dr
		} else {
			parent = directories[parents[id]]
		}
		dir.parent = parent
		parent.SetContent(name, dir)
	}
	return directories, nil
}

func fetchFiles(db *sql.DB, dr *drive, directories map[int]*vfs_dir) error {
	stmt, err := db.Prepare("SELECT id, name, directoryId, uuid, created, updated, metadata FROM files WHERE driveId=?")
	if err != nil {
		return fmt.Errorf("db.Prepare(files): %w", err)
	}
	rows, err := stmt.Query(dr.id)
	if err != nil {
		return fmt.Errorf("stmt.Query(files): %w", err)
	}
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
			return fmt.Errorf("error reading files table: %w", err)
		}
		upTime := time.Unix(updated, 0)
		crTime := time.Unix(created, 0)
		var dir VirtualFS
		if directoryId < 0 {
			dir = dr
		} else {
			dir = directories[directoryId]
		}
		file := &vfs_file{name, uuid, dir, crTime, upTime, metadata, id}
		dir.SetContent(name, file)
	}
	return nil
}

func fetchDrives(root Root) (map[string]Drive, error) {
	
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	
	drives:= make(map[string]Drive)
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
		var store storage.Storage
		if host == "gcs" {
			store = storage.NewGoogleCloud(address)
		} else if host == "local" {
			store = storage.NewLocalFileSystem(address)
		} else {
			// Unknown type - skip silently.
			continue
		}
		drives[name] = &drive{name, description, "", id, store, nil, root.AsVirtualFS()}
	}
	db.Close()
	return drives, nil
}

func createCatalogFile(fileObj *vfs_file) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	driveId := fileObj.Drive().CatalogId()
	if fileObj.Parent() == nil {
		return fmt.Errorf("creating file at root level")
	}
	parentId := fileObj.Parent().CatalogId()
	if parDrive := fileObj.Parent().AsDrive(); parDrive != nil {
		// If parent is a drive, then parentId must be set to -1
		parentId = -1
	}

	stmt, err := db.Prepare("INSERT INTO files (driveId, name, directoryId, uuid, created, updated, metadata) values (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("db.Prepare: %w", err)
	}
	
	if _, err := stmt.Exec(driveId, fileObj.name, parentId, fileObj.uuid, fileObj.created.Unix(), fileObj.updated.Unix(), fileObj.metadata); err != nil {
		return fmt.Errorf("stmt.Exec: %w", err)
	}
	rows, err := db.Query("SELECT last_insert_rowid()")
	if err != nil {
		return fmt.Errorf("db.Query %w", err)
	}
	var id int64
	rows.Next()
	rows.Scan(&id)
	fileObj.id = int(id)
	db.Close()
	return nil
}

func createCatalogDirectory(dirObj *vfs_dir) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	driveId := dirObj.Drive().CatalogId()
	if dirObj.Parent() == nil {
		return fmt.Errorf("creating directory at root level")
	}
	parentId := dirObj.Parent().CatalogId()
	if dirObj.Parent().IsDrive() {
		// If parent is a drive, then parentId must be set to -1
		parentId = -1
	}

	stmt, err := db.Prepare("INSERT INTO directories (driveId, name, parentId) values (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("db.Prepare: %w", err)
	}
	if _, err := stmt.Exec(driveId, dirObj.name, parentId); err != nil {
		return fmt.Errorf("stmt.Exec: %w", err)
	}
	rows, err := db.Query("SELECT last_insert_rowid()")
	if err != nil {
		return fmt.Errorf("db.Query %w", err)
	}
	var id int64
	rows.Next()
	rows.Scan(&id)
	dirObj.id = int(id)
	db.Close()
	return nil
}

func updateCatalogFile(id int, name string, parent VirtualFS, updated time.Time) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	parentId := parent.CatalogId()
	if parent.IsDrive() {
		// If parent is a drive, then parentId must be set to -1
		parentId = -1
	}

	stmt, err := db.Prepare("UPDATE files SET name = ?, directoryId = ?, updated = ? where id = ?")
	if err != nil {
		return fmt.Errorf("db.Prepare: %w", err)
	}
	
	if _, err := stmt.Exec(name, parentId, updated.Unix(), id); err != nil {
		return fmt.Errorf("stmt.Exec: %w", err)
	}
	db.Close()
	return nil
}

func updateCatalogDirectory(id int, name string, parent VirtualFS) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	parentId := parent.CatalogId()
	if parent.IsDrive() {
		// If parent is a drive, then parentId must be set to -1
		parentId = -1
	}

	stmt, err := db.Prepare("UPDATE directories SET name = ?, parentId = ? where id = ?")
	if err != nil {
		return fmt.Errorf("db.Prepare: %w", err)
	}
	
	if _, err := stmt.Exec(name, parentId, id); err != nil {
		return fmt.Errorf("stmt.Exec: %w", err)
	}
	db.Close()
	return nil
}
