
package virtualfs

import (
	"fmt"
)

type vfs_dir struct {
	name string
	path string
	content map[string]VirtualFS
	parent VirtualFS
}

func (d *vfs_dir) IsFile() bool {
	return false
}

func (d *vfs_dir) AsFile() File {
	return nil
}

func (d *vfs_dir) IsDir() bool {
	return true
}

func (d *vfs_dir) AsDrive() Drive {
	return nil
}

func (d *vfs_dir) Drive() Drive {
	return findDrive(d)
}

func (d *vfs_dir) Name() string {
	return d.name
}

func (d *vfs_dir) LocalPath() string {
	return d.path
}

func (d *vfs_dir) FullPath() string {
	return fmt.Sprintf("/%s%s", d.Drive().Name(), d.path)
}

func (d *vfs_dir) Parent() VirtualFS {
	return d.parent
}

func (d *vfs_dir) ContentList() []string {
	result := make([]string, 0, len(d.content))
	for k, _ := range d.content {
		result = append(result, k)
	}
	return result
}

func (d *vfs_dir) GetContent(field string) (VirtualFS, bool) {
	result, found := d.content[field]
	return result, found
}

func (d *vfs_dir) SetContent(field string, value VirtualFS) {
	d.content[field] = value
}

func (d *vfs_dir) Print() {
	fmt.Printf("Name:     %s\n", d.name)
	fmt.Printf("Path:     %s\n", d.path)
}

func (d *vfs_dir) Root() VirtualFS {
	return findRoot(d)
}

