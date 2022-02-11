
package virtualfs

import (
	"fmt"
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
