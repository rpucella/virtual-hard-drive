
package catalog

import (
	"fmt"
	"strings"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type VFS interface {     // VFS = Virtual File System
	AsFile() (File, bool)
	AsDir() (Directory, bool)
	AsRoot() (Root, bool)
	AsDrive() (Drive, bool)
	Name() string
	Path() string
	Parent() VFS
	Root() Root
	Print() 
}

type Root interface {
	Name() string
	Path() string
	Parent() Root
	Content() []Drive
}

type Drive interface {
	Name() string
//	Print()
//	Content() []Catalog
	Description() string
	Storage() storage.Storage
	FetchCatalog() (Catalog, error)
	UpdateCatalog(Catalog) error
}

type Catalog interface {
	IsFile() bool
	IsDir() bool
	Name() string
	Path() string
	Parent() Catalog
	Root() Catalog
	Print()
	UUID() string
	Content() map[string]Catalog
	SetContent(string, Catalog)
}

type Directory struct {
	name string
	path string
	content map[string]Catalog
	parent Catalog
}

type File struct {
	name string
	path string
	uuid string
	parent Catalog
}

func NewCatalog(flat []byte) (Catalog, error) {
	// Convert to a string first.
	strFlat := string(flat)
	///fmt.Printf("Flat: [%s]\n", strFlat)
	lines := strings.Split(strFlat, "\n")
	var cat Catalog = &Directory{"", "/", make(map[string]Catalog), nil}
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if len(line) > 0 {
			// Skip empty lines.
			path, uuid, err := splitLine(line)
			if err != nil {
				return nil, fmt.Errorf("cannot parse catalog: %w", err)
			}
			if uuid == "" {
				// Directory only.
				directories, err := splitPath(path)
				if err != nil {
					return nil, fmt.Errorf("cannot parse catalog: %w", err)
				}
				if _, err := walkCreateDirectories(cat, path, directories); err != nil {
					return nil, fmt.Errorf("cannot parse catalog: %w", err)
				}
			} else { 
				directories, file, err := splitPathFile(path)
				if err != nil {
					return nil, fmt.Errorf("cannot parse catalog: %w", err)
				}
				curr, err := walkCreateDirectories(cat, path, directories)
				currPath := curr.Path()
				// At this point, curr is in the directory where we want the file.
				if curr.IsFile() {
					return nil, fmt.Errorf("file in middle of path %s", path)
				}
				content := curr.Content()
				_, exists := content[file]
				if exists {
					return nil, fmt.Errorf("file %s already exists in path %s", file, path)
				}
				fileObj := &File{file, currPath + file, uuid, curr}
				curr.SetContent(file, fileObj)
			}
		}
	}
	return cat, nil
}

func walkCreateDirectories(cat Catalog, path string, directories []string) (Catalog, error) {
	// We can probably merge this function with NavigateCreateLast() below
	var curr Catalog
	var currPath string
	for i, dir := range directories {
		if curr == nil {
			// First directory should be empty
			if i != 0 || dir != "" {
				return nil, fmt.Errorf("path should be absolute %s", path)
			}
			curr = cat
			currPath = "/"
		} else if curr.IsFile() {
			return nil, fmt.Errorf("file in middle of path %s", path)
		} else if curr.IsDir() {
			currPath = currPath + dir + "/"
			// does the name exist?
			content := curr.Content()
			dirObj, ok := content[dir]
			if ok {
				curr = dirObj
			} else {
				// Need to create the directory!
				dirObj = &Directory{dir, currPath, make(map[string]Catalog), curr}
				curr.SetContent(dir, dirObj)
				curr = dirObj
			}
		} else {
			return nil, fmt.Errorf("unknown catalog object %v", curr)
		}
	}
	return curr, nil
}

func splitLine(line string) (string, string, error) {
	ss := strings.Split(line, ":")
	if len(ss) < 1 || len(ss) > 2 {
		return "", "", fmt.Errorf("wrong number of fields in line %d", len(ss))
	}
	if len(ss) == 1 {
		// Directory.
		return ss[0], "", nil
	}
	// File.
	return ss[0], ss[1], nil
}

func splitPathFile(path string) ([]string, string, error) {
	ss := strings.Split(path, "/")
	if len(ss) < 1 {
		return nil, "", fmt.Errorf("malformed path %s", path)
	}
	return ss[:len(ss) - 1], ss[len(ss) - 1], nil
}

func splitPath(path string) ([]string, error) {
	return strings.Split(path, "/"), nil
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

func (d *Directory) Path() string {
	return d.path
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

func (d *Directory) Print() {
	fmt.Printf("Name:     %s\n", d.name)
	fmt.Printf("Path:     %s\n", d.path)
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

func (f *File) Path() string {
	return f.path
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

func (f *File) Print() {
	fmt.Printf("Name:     %s\n", f.name)
	fmt.Printf("Path:     %s\n", f.path)
	fmt.Printf("UUID:     %s\n", f.uuid)
}


func findRoot(cat Catalog) Catalog {
	var curr Catalog = cat
	for {
		if curr.Parent() == nil {
			return curr
		}
		curr = curr.Parent()
	}
}

func (d *Directory) Root() Catalog {
	return findRoot(d)
}

func (f *File) Root() Catalog {
	return findRoot(f)
}

func DecomposePath(path string) []string {
	return strings.Split(path, "/")
}

func DecomposePathFile(path string) ([]string, string) {
	content := strings.Split(path, "/")
	if len(content) == 0 {
		return content, ""
	}
	return content[:len(content) - 1], content[len(content) - 1]
}

func Navigate(cat Catalog, path string) (Catalog, error) {
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
	}
	dirs := DecomposePath(cleanPath)
	if len(dirs) == 0 {
		return nil, fmt.Errorf("empty path to navigate")
	}
	var curr Catalog = cat
	for _, dir := range dirs {
		if dir == "" {
			// Reset to root!
			curr = findRoot(curr)
		} else if dir == "." {
			// Do nothing!
		} else if dir == ".." {
			if curr.Parent() == nil {
				return nil, fmt.Errorf("root has no parent")
			}
			curr = curr.Parent()
		} else {
			newCurr, found := curr.Content()[dir]
			if !found {
				return nil, fmt.Errorf("cannot find folder: %s", dir)
			}
			if !newCurr.IsDir() {
				return nil, fmt.Errorf("not a folder: %s", newCurr.Name())
			}
			curr = newCurr
		}
	}
	return curr, nil
}

func NavigateCreateLast(cat Catalog, path string) error {
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
	}
	dirs := DecomposePath(cleanPath)
	if len(dirs) == 0 {
		return fmt.Errorf("empty path to navigate")
	}
	lastDir := dirs[len(dirs) - 1]
	if lastDir == "." || lastDir == ".." {
		return fmt.Errorf("cannot create . or ..")
	}
	dirs = dirs[:len(dirs) - 1]
	var curr Catalog = cat
	for _, dir := range dirs {
		if dir == "" {
			// Reset to root!
			curr = findRoot(curr)
		} else if dir == "." {
			// Do nothing!
		} else if dir == ".." {
			if curr.Parent() == nil {
				return fmt.Errorf("root has no parent")
			}
			curr = curr.Parent()
		} else {
			newCurr, found := curr.Content()[dir]
			if !found {
				return fmt.Errorf("cannot find folder: %s", dir)
			}
			if !newCurr.IsDir() {
				return fmt.Errorf("not a folder: %s", newCurr.Name())
			}
			curr = newCurr
		}
	}
	// It better not exist.
	if _, found := curr.Content()[lastDir]; found {
		return fmt.Errorf("folder already exists: %s", lastDir)
	}
	dirObj := &Directory{lastDir, curr.Path() + lastDir + "/", make(map[string]Catalog), curr}
	curr.SetContent(lastDir, dirObj)
	return nil
}	


func NavigateFile(cat Catalog, path string) (Catalog, error) {
	dirs, file := DecomposePathFile(path)
	var curr Catalog = cat
	for _, dir := range dirs {
		if dir == "" {
			// Reset to root!
			curr = findRoot(curr)
		} else if dir == "." {
			// Do nothing!
		} else if dir == ".." {
			if curr.Parent() == nil {
				return nil, fmt.Errorf("root has no parent")
			}
			curr = curr.Parent()
		} else {
			newCurr, found := curr.Content()[dir]
			if !found {
				return nil, fmt.Errorf("cannot find folder: %s", dir)
			}
			if !newCurr.IsDir() {
				return nil, fmt.Errorf("not a folder: %s", dir)
			}
			curr = newCurr
		}
	}
	fileObj, found := curr.Content()[file]
	if !found {
		return nil, fmt.Errorf("cannot find file: %s", file)
	}
	if !fileObj.IsFile() {
		return nil, fmt.Errorf("not a file: %s", file)
	}
	return fileObj, nil
}

func AddFile(cat Catalog, name string, uuid string) error {
	_, found := cat.Content()[name]
	if found {
		return fmt.Errorf("file %s already exists at %s", name, cat.Path())
	}
	path := cat.Path() + name
	file := &File{name, path, uuid, cat}
	cat.Content()[name] = file
	return nil
}

func flatten(cat Catalog, prefix string) []string {
	result := make([]string, 0)     // cat.Size()
	if cat == nil { 
		return result
	}
	if cat.IsFile() {
		line := fmt.Sprintf("%s:%s", prefix, cat.UUID())
		result = append(result, line)
	} else {
		if prefix != "" {
			line := fmt.Sprintf("%s", prefix)
			result = append(result, line)
		}
		for k, v := range cat.Content() {
			newPrefix := fmt.Sprintf("%s/%s", prefix, k)
			newLines := flatten(v, newPrefix)
			for _, line := range newLines {
				result = append(result, line)
			}
		}
	}
	return result
}

func Flatten(cat Catalog) []string {
	return flatten(cat, "")
}
