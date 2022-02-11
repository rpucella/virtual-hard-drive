
package catalog

import (
	"fmt"
)

type root struct {
	drives map[string]Drive
}

func (r *root) Drives() map[string]Drive {
	return r.drives
}

func (r *root) AsCatalog() Catalog {
	return r
}

func (r *root) IsFile() bool {
	return false
}

func (r *root) IsDir() bool {
	return true
}

func (r *root) Name() string {
	return ""
}

func (r *root) LocalPath() string {
	return "/"
}

func (r *root) FullPath() string {
	return "/"
}

func (r *root) Parent() Catalog {
	return nil
}

func (r *root) Root() Catalog {
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

func (r *root) UUID() string {
	return ""
}

func (r *root) ContentList() []string {
	result := make([]string, 0, len(r.drives))
	for k, _ := range r.drives {
		result = append(result, k)
	}
	return result
}

func (r *root) GetContent(field string) (Catalog, bool) {
	result, found := r.drives[field]
	if !found {
		return nil, false
	}
	return result.AsCatalog(), true
}

func (r *root) SetContent(name string, value Catalog) {
	// Do nothing silently?
	return
}

func NewRoot() (Root, error) {
	root := &root{}
	content, err := readDrives(root)
	if err != nil {
		return nil, fmt.Errorf("cannot read drives: %w", err)
	}
	root.drives = content
	return root, nil 
}
