
package virtualfs

import (
	"fmt"
	"time"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
	"rpucella.net/virtual-hard-drive/internal/catalog"
)

type drive struct{
	name string
	description string
	id int                  // Identifier in catalog.db.
	catalog catalog.Catalog              
	storage storage.Storage
	top VirtualFS           // This is a horrible name.
	root VirtualFS
	// Add possible restriction flags (i.e., warn in case of too recent deletes, etc)
}

func (d *drive) Name() string {
	return d.name
}

func (d *drive) Description() string {
	return d.description
}

func (d *drive) Storage() storage.Storage {
	return d.storage
}

func (d *drive) AsVirtualFS() VirtualFS {
	return d
}

func (d *drive) AsDrive() Drive {
	return d
}

func (r *drive) IsFile() bool {
	return false
}

func (r *drive) AsFile() File {
	return nil
}

func (r *drive) AsDir() Directory {
	return r
}

func (r *drive) IsDir() bool {
	return true
}

func (r *drive) IsRoot() bool {
	return false
}

func (r *drive) IsDrive() bool {
	return true
}

func (r *drive) Path() string {
	return constructPath(r) + "/"
}

func (r *drive) Parent() VirtualFS {
	return r.root
}

func (r *drive) Root() VirtualFS {
	return r.root
}

func (r *drive) Drive() Drive {
	return r
}

func (r *drive) CatalogId() int {
	return r.id
}

func (r *drive) Print() {
	// TODO: Complete.
	fmt.Printf("<Drive %s>\n", r.name)
}

func (r *drive) ContentList() []string {
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	return r.top.ContentList()
}

func (r *drive) GetContent(field string) (VirtualFS, bool) {
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	result, found := r.top.GetContent(field)
	return result, found
}

func (r *drive) SetContent(name string, value VirtualFS) {
	// Do nothing silently?
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	r.top.SetContent(name, value)
}

func (r *drive) DelContent(name string) {
	// Do nothing silently?
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	r.top.DelContent(name)
}

func (r *drive) Move(targetDir VirtualFS, name string) error {
	return fmt.Errorf("cannot move drive")
}

func fetchCatalog(r *drive) error {
	directories, err := r.catalog.FetchDirectories(r.id)
	if err != nil {
		return err
	}
	files, err := r.catalog.FetchFiles(r.id)
	if err != nil {
		return err
	}
	
	dirMap := make(map[int]*vfs_dir)
	parentMap := make(map[int]int)
	for id, dir := range directories {
		dirMap[id] = &vfs_dir{dir.Name, make(map[string]VirtualFS), nil, id}
		parentMap[id] = dir.ParentId
	}
	r.top = &vfs_dir{"", make(map[string]VirtualFS), r.root, -1}
	for _, dir := range dirMap {
		name := dir.name
		var parent VirtualFS
		if parentMap[dir.id] < 0 {
			parent = r
		} else {
			parent = dirMap[parentMap[dir.id]]
		}
		dir.parent = parent
		parent.SetContent(name, dir)
	}
	for id, file := range files {
		name := file.Name
		var dir VirtualFS
		if file.DirectoryId < 0 {
			dir = r
		} else {
			dir = dirMap[file.DirectoryId]
		}
		fileObj := &vfs_file{name, file.UUID, dir, file.Created, file.Updated, file.Metadata, id}
		dir.SetContent(name, fileObj)
	}
	return nil
}

func (r *drive) createFile(name string, uuid string, dirId int, created time.Time, updated time.Time, metadata string) (int, error) {
	fileId, err := r.catalog.CreateFile(r.id, name, uuid, dirId, created, updated, metadata)
	if err != nil {
		return 0, err
	}
	return fileId, nil
}

func (r *drive) createDirectory(name string, parentId int) (int, error) {
	dirId, err := r.catalog.CreateDirectory(r.id, name, parentId)
	if err != nil {
		return 0, err
	}
	return dirId, nil
}

func (r *drive) updateFile(id int, name string, dirId int) error {
	err := r.catalog.UpdateFile(id, name, dirId)
	return err
}

func (r *drive) updateDirectory(id int, name string, parentId int) error {
	err := r.catalog.UpdateDirectory(id, name, parentId)
	return err
}

func (r *drive) countFilesInDir(dirId int) (int, error) {
	count, err := r.catalog.CountFilesInDirectory(dirId)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *drive) CountFiles() (int, error) {
	count, err := d.catalog.CountFilesInDrive(d.id)
	if err != nil {
		return 0, err
	}
	return count, nil
}
