
package catalog

import (
	"fmt"
	"strings"
)

type Catalog interface {
	IsFile() bool
	IsDir() bool
	Name() string
	Parent() Catalog
	Content() map[string]Catalog
	SetContent(string, Catalog)
	UUID() string
}

type Directory struct {
	name string
	content map[string]Catalog
	parent Catalog
}

type File struct {
	name string
	uuid string
	parent Catalog
}

func NewCatalog(flat []byte) (Catalog, error) {
	// Convert to a string first.
	strFlat := string(flat)
	///fmt.Printf("Flat: [%s]\n", strFlat)
	lines := strings.Split(strFlat, "\n")
	var cat Catalog = &Directory{"", make(map[string]Catalog), nil}
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if len(line) > 0 {
			// Skip empty lines.
			path, uuid, err := splitLine(line)
			if err != nil {
				return nil, fmt.Errorf("cannot parse catalog: %w", err)
			}
			directories, file, err := splitPath(path)
			if err != nil {
				return nil, fmt.Errorf("cannot parse catalog: %w", err)
			}
			var curr Catalog = nil
			for i, dir := range directories {
				if curr == nil {
					// First directory should be empty
					if i != 0 || dir != "" {
						return nil, fmt.Errorf("path should be absolute %s", path)
					}
					curr = cat
				} else if curr.IsFile() {
					return nil, fmt.Errorf("file in middle of path %s", path)
				} else if curr.IsDir() {
					// does the name exist?
					content := curr.Content()
					dirObj, ok := content[dir]
					if ok {
						curr = dirObj
					} else {
						// Need to create the directory!
						dirObj = &Directory{dir, make(map[string]Catalog), curr}
						curr.SetContent(dir, dirObj)
						curr = dirObj
					}
				} else {
					return nil, fmt.Errorf("unknown catalog object %v", curr)
				}
			}
			// At this point, curr is in the directory where we want the file.
			if curr.IsFile() {
				return nil, fmt.Errorf("file in middle of path %s", path)
			}
			content := curr.Content()
			_, exists := content[file]
			if exists {
				return nil, fmt.Errorf("file %s already exists in path %s", file, path)
			}
			fileObj := &File{file, uuid, curr}
			curr.SetContent(file, fileObj)
		}
	}
	return cat, nil
}

func splitLine(line string) (string, string, error) {
	ss := strings.Split(line, ":")
	if len(ss) != 2 {
		return "", "", fmt.Errorf("wrong number of fields in line %d", len(ss))
	}
	return ss[0], ss[1], nil
}

func splitPath(path string) ([]string, string, error) {
	ss := strings.Split(path, "/")
	if len(ss) < 1 {
		return nil, "", fmt.Errorf("malformed path %s", path)
	}
	return ss[:len(ss) - 1], ss[len(ss) - 1], nil
}

func spaces(n int) string {
	return strings.Repeat(" ", n)
}

func printLevel(curr Catalog, indent int) {
	if curr.IsFile() {
		fmt.Printf("%s%s\n", spaces(indent), curr.Name())
	} else if curr.IsDir() {
		fmt.Printf("%s%s/\n", spaces(indent), curr.Name())
		for _, sub := range curr.Content() {
			printLevel(sub, indent + 2)
		}
	}
}

func Print(cat Catalog) {
	printLevel(cat, 0)
}

func (d *Directory) IsFile() bool {
	return false
}

func (d *Directory) IsDir() bool {
	return true
}

func (d *Directory) Name() string {
	return d.name
}

func (d *Directory) Parent() Catalog {
	return d.parent
}

func (d *Directory) Content() map[string]Catalog {
	return d.content
}

func (d *Directory) SetContent(field string, value Catalog) {
	d.content[field] = value
}

func (d *Directory) UUID() string {
	return ""
}

func (f *File) IsFile() bool {
	return true
}

func (f *File) IsDir() bool {
	return false
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Parent() Catalog {
	return f.parent
}

func (f *File) Content() map[string]Catalog {
	return nil
}

func (f *File) SetContent(field string, value Catalog) {
	// Mmm. Do nothing.
}

func (f *File) UUID() string {
	return f.uuid
}

