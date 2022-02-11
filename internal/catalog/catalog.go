
package catalog

import (
	"fmt"
	"strings"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type Root interface {
	Drives() map[string]Drive
	AsCatalog() Catalog
}

type Drive interface {
	Name() string
	Description() string
	Storage() storage.Storage
	UpdateCatalog() error
	AsCatalog() Catalog
}

type Catalog interface {
	IsFile() bool
	IsDir() bool
	AsDrive() Drive
	Name() string
	LocalPath() string   // This is the local path of the entry.
	FullPath() string    // Full path including drive name.    
	Parent() Catalog
	Root() Catalog
	Drive() Drive        // Returns the drive of this node (if any).
	Print()
	UUID() string
	ContentList() []string
	GetContent(string) (Catalog, bool)
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
			dirObj, ok := curr.GetContent(dir)
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
		for _, k := range curr.ContentList() {
			sub, _ := curr.GetContent(k)
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

func (d *Directory) AsDrive() Drive {
	return nil
}

func (d *Directory) Drive() Drive {
	return findDrive(d)
}

func (d *Directory) Name() string {
	return d.name
}

func (d *Directory) LocalPath() string {
	return d.path
}

func (d *Directory) FullPath() string {
	return fmt.Sprintf("/%s%s", d.Drive().Name(), d.path)
}

func (d *Directory) Parent() Catalog {
	return d.parent
}

func (d *Directory) ContentList() []string {
	result := make([]string, 0, len(d.content))
	for k, _ := range d.content {
		result = append(result, k)
	}
	return result
}

func (d *Directory) GetContent(field string) (Catalog, bool) {
	result, found := d.content[field]
	return result, found
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

func (f *File) AsDrive() Drive {
	return nil
}

func (f *File) Drive() Drive {
	return findDrive(f)
}

func (f *File) Name() string {
	return f.name
}

func (f *File) LocalPath() string {
	return f.path
}

func (f *File) FullPath() string {
	return fmt.Sprintf("/%s%s", f.Drive().Name(), f.path)
}

func (f *File) Parent() Catalog {
	return f.parent
}

func (f *File) ContentList() []string {
	return nil
}

func (f *File) GetContent(field string) (Catalog, bool) {
	return nil, false
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

func findDrive(cat Catalog) Drive {
	curr := cat
	for {
		if curr == nil {
			return nil
		}
		if drive := curr.AsDrive(); drive != nil {
			return drive
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
			newCurr, found := curr.GetContent(dir)
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

func NavigateCreateLast(cat Catalog, path string) (Catalog, error) {
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
	}
	dirs := DecomposePath(cleanPath)
	if len(dirs) == 0 {
		return nil, fmt.Errorf("empty path to navigate")
	}
	lastDir := dirs[len(dirs) - 1]
	if lastDir == "." || lastDir == ".." {
		return nil, fmt.Errorf("cannot create . or ..")
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
				return nil, fmt.Errorf("root has no parent")
			}
			curr = curr.Parent()
		} else {
			newCurr, found := curr.GetContent(dir)
			if !found {
				return nil, fmt.Errorf("cannot find folder: %s", dir)
			}
			if !newCurr.IsDir() {
				return nil, fmt.Errorf("not a folder: %s", newCurr.Name())
			}
			curr = newCurr
		}
	}
	// It better not exist.
	if _, found := curr.GetContent(lastDir); found {
		return nil, fmt.Errorf("folder already exists: %s", lastDir)
	}
	dirObj := &Directory{lastDir, curr.LocalPath() + lastDir + "/", make(map[string]Catalog), curr}
	curr.SetContent(lastDir, dirObj)
	return dirObj, nil
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
			newCurr, found := curr.GetContent(dir)
			if !found {
				return nil, fmt.Errorf("cannot find folder: %s", dir)
			}
			if !newCurr.IsDir() {
				return nil, fmt.Errorf("not a folder: %s", dir)
			}
			curr = newCurr
		}
	}
	fileObj, found := curr.GetContent(file)
	if !found {
		return nil, fmt.Errorf("cannot find file: %s", file)
	}
	if !fileObj.IsFile() {
		return nil, fmt.Errorf("not a file: %s", file)
	}
	return fileObj, nil
}

func AddFile(cat Catalog, name string, uuid string) error {
	_, found := cat.GetContent(name)
	if found {
		return fmt.Errorf("file %s already exists at %s", name, cat.FullPath())
	}
	path := cat.LocalPath() + name
	file := &File{name, path, uuid, cat}
	cat.SetContent(name, file)
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
		for _, k := range cat.ContentList() {
			v, _ := cat.GetContent(k)
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
