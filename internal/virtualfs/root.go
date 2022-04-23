
package virtualfs

import (
	"fmt"
	"rpucella.net/virtual-hard-drive/internal/catalog"
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type root struct {
	drives map[string]Drive
}

func (r *root) Drives() map[string]Drive {
	return r.drives
}

func (r *root) AsVirtualFS() VirtualFS {
	return r
}

func (r *root) IsFile() bool {
	return false
}

func (r *root) AsFile() File {
	return nil
}

func (r *root) AsDir() Directory {
	return nil
}

func (r *root) IsDir() bool {
	return true
}

func (r *root) IsRoot() bool {
	return true
}

func (r *root) IsDrive() bool {
	return false
}

func (r *root) Name() string {
	return ""
}

func (r *root) Path() string {
	return "/"
}

func (r *root) Parent() VirtualFS {
	return nil
}

func (r *root) Root() VirtualFS {
	return r
}

func (r *root) Drive() Drive {
	return nil
}

func (r *root) AsDrive() Drive {
	return nil
}

func (r *root) Print() {
	fmt.Println("<Root>")
}

func (r *root) ContentList() []string {
	result := make([]string, 0, len(r.drives))
	for k, _ := range r.drives {
		result = append(result, k)
	}
	return result
}

func (r *root) GetContent(field string) (VirtualFS, bool) {
	result, found := r.drives[field]
	if !found {
		return nil, false
	}
	return result.AsVirtualFS(), true
}

func (r *root) SetContent(name string, value VirtualFS) {
	// Do nothing.
}

func (r *root) DelContent(name string) {
	// Do nothing.
}

func (r *root) CatalogId() int {
	return -1
}

func (r *root) CountFiles() (int, error) {
	return 0, nil
}

func NewRoot(c catalog.Catalog) (Root, error) {
	root := &root{}
	content, err := c.FetchDrives()
	if err != nil {
		return nil, fmt.Errorf("cannot read drives: %w", err)
	}
	root.drives = make(map[string]Drive)
	for _, driveDesc := range content {
		var store storage.Storage
		if driveDesc.Type == "gcs" {
			newStore, err := storage.NewGoogleCloud(driveDesc.Location)
			if err != nil {
				return nil, fmt.Errorf("cannot connect to GCS: %w", err)
			}
			store = newStore
		} else if driveDesc.Type == "local" {
			store = storage.NewLocalFileSystem(driveDesc.Location)
		} else {
			// Unknown type - skip silently.
			continue
		}
		root.drives[driveDesc.Name] = &drive{
			driveDesc.Name,
			driveDesc.Description,
			driveDesc.Id,
			c,
			store,
			nil,
			root.AsVirtualFS(),
		}
	}
	return root, nil 
}

func (r *root) Move(targetDir VirtualFS, name string) error {
	return fmt.Errorf("cannot move root")
}
