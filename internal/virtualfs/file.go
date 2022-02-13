
package virtualfs

import (
	"fmt"
	"time"
)

type vfs_file struct {
	name string
	uuid string
	parent VirtualFS
	created time.Time
	updated time.Time
	metadata string
	id int              // Identifier in catalog.db.
}

func (f *vfs_file) IsFile() bool {
	return true
}

func (f *vfs_file) AsFile() File {
	return f
}

func (f *vfs_file) IsDir() bool {
	return false
}

func (f *vfs_file) AsDrive() Drive {
	return nil
}

func (f *vfs_file) Drive() Drive {
	return findDrive(f)
}

func (f *vfs_file) Name() string {
	return f.name
}

func (f *vfs_file) Path() string {
	return constructPath(f)
}

func (f *vfs_file) Parent() VirtualFS {
	return f.parent
}

func (f *vfs_file) ContentList() []string {
	return nil
}

func (f *vfs_file) GetContent(field string) (VirtualFS, bool) {
	return nil, false
}

func (f *vfs_file) SetContent(field string, value VirtualFS) {
	// Do nothing.
}

func (f *vfs_file) DelContent(field string) {
	// Do nothing.
}

func (f *vfs_file) UUID() string {
	return f.uuid
}

func (f *vfs_file) CatalogId() int {
	return f.id
}

func (f *vfs_file) Print() {
	fmt.Println()
	fmt.Printf("Name:       %s\n", f.name)
	fmt.Printf("Path:       %s\n", f.Path())
	fmt.Printf("UUID:       %s\n", f.uuid)
	fmt.Printf("Created     %s\n", f.created.Format(time.RFC822))
	fmt.Printf("Updated:    %s\n", f.updated.Format(time.RFC822))
	fmt.Printf("Metadata:   %s\n", f.metadata)
	fmt.Printf("Catalog ID: %d\n", f.id)
}

func (f *vfs_file) Root() VirtualFS {
	return findRoot(f)
}

func (f *vfs_file) Created() time.Time {
	return f.created
}

func (f *vfs_file) Updated() time.Time {
	return f.updated
}

func (f *vfs_file) Metadata() string {
	return f.metadata
}
