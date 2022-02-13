
package virtualfs

import (
	"fmt"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type drive struct{
	name string
	description string
	catalogPath string      // This could be kept private.
	id int                  // Identifier in catalog.db.
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

func (r *drive) IsDir() bool {
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
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	return r.top.ContentList()
}

func (r *drive) GetContent(field string) (VirtualFS, bool) {
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	result, found := r.top.GetContent(field)
	return result, found
}

func (r *drive) SetContent(name string, value VirtualFS) {
	// Do nothing silently?
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	r.top.SetContent(name, value)
}

func (r *drive) DelContent(name string) {
	// Do nothing silently?
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	r.top.DelContent(name)
}
