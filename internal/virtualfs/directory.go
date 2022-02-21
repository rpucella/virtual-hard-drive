
package virtualfs

import (
	"fmt"
)

type vfs_dir struct {
	name string
	content map[string]VirtualFS
	parent VirtualFS
	id int              // Identifier in catalog.db.
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

func (d *vfs_dir) IsRoot() bool {
	return false
}

func (d *vfs_dir) IsDrive() bool {
	return false
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

func (d *vfs_dir) Path() string {
	return constructPath(d) + "/"
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

func (d *vfs_dir) DelContent(field string) {
	delete(d.content, field)
}

func (d *vfs_dir) CatalogId () int {
	return d.id
}

func (d *vfs_dir) Print() {
	fmt.Printf("Name:       %s\n", d.name)
	fmt.Printf("Path:       %s\n", d.Path())
	fmt.Printf("Catalog ID: %d\n", d.id)
}

func (d *vfs_dir) Root() VirtualFS {
	return findRoot(d)
}

func (d *vfs_dir) Move(targetDir VirtualFS, name string) error {
	// Move to `targetDir` under name `name`.
	if err := ValidateName(name); err != nil {
		return err
	}
	if _, found := targetDir.GetContent(name); found {
		return fmt.Errorf("name %s already exists in %s", name, targetDir.Name())
	}
	if targetDir.IsRoot() {
		return fmt.Errorf("cannot move directory to root")
	}
	if d.Drive() != targetDir.Drive() {
		return fmt.Errorf("cannot move directory across drives")
	}
	// Also check that the source directory is not an ancestor of the target directory!
	curr := targetDir
	for !curr.IsDrive() {
		// Can actually stop looking when the current folder is the drive
		// since we can't move the drive per above.
		if curr == d {
			return fmt.Errorf("trying to move directory to a descendant")
		}
		curr = curr.Parent()
	}
	parentId := targetDir.CatalogId()
	if targetDir.IsDrive() {
		// Override if we're putting it in a drive
		parentId = -1
	}
	if err := d.Drive().updateDirectory(d.id, name, parentId); err != nil {
		return err
	}
	// If update was successful, update the tree.
	d.parent.DelContent(d.name)
	d.parent = targetDir
	d.name = name
	targetDir.SetContent(name, d)
	return nil
}
