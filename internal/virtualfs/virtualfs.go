
package virtualfs

import (
	"fmt"
	"strings"
	"time"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

type Root interface {
	Drives() map[string]Drive
	AsVirtualFS() VirtualFS
}

type Drive interface {
	Name() string
	Description() string
	Storage() storage.Storage
	AsVirtualFS() VirtualFS
	CatalogId() int
}

type File interface {
	Name() string
	UUID() string
	Created() time.Time
	Updated() time.Time
	Metadata() string       // Storage-specific metadata (such as # of chunks),
}

type VirtualFS interface {
	IsFile() bool
	IsDir() bool
	AsDrive() Drive
	AsFile() File
	Name() string
	Path() string        // Full path including drive name.    
	Parent() VirtualFS
	Root() VirtualFS
	Drive() Drive        // Returns the drive of this node (if any).
	Print()
	ContentList() []string
	GetContent(string) (VirtualFS, bool)
	SetContent(string, VirtualFS)
	DelContent(string)
	CatalogId() int      // Meaning depends on the kind of virtual FS node we have.
}

func constructPath(vfs VirtualFS) string {
	path := make([]string, 0, 10)
	curr := vfs
	for curr != nil {
		path = append(path, curr.Name())
		curr = curr.Parent()
	}
	// Reverse
	for i, j := 0, len(path) - 1; i < j; i, j = i + 1, j - 1 {
		path[i], path[j] = path[j], path[i]
	}
	return strings.Join(path, "/")
}

func spaces(n int) string {
	return strings.Repeat(" ", n)
}

func printLevel(curr VirtualFS, indent int) {
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

func Print(cat VirtualFS) {
	printLevel(cat, 0)
}

func findRoot(cat VirtualFS) VirtualFS {
	var curr VirtualFS = cat
	for {
		if curr.Parent() == nil {
			return curr
		}
		curr = curr.Parent()
	}
}

func findDrive(cat VirtualFS) Drive {
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

func Navigate(cat VirtualFS, path string) (VirtualFS, error) {
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
	}
	dirs := DecomposePath(cleanPath)
	if len(dirs) == 0 {
		return nil, fmt.Errorf("empty path to navigate")
	}
	var curr VirtualFS = cat
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

func NavigateCreateLast(cat VirtualFS, path string) (VirtualFS, error) {
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
	var curr VirtualFS = cat
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
	dirObj, err := CreateDirectory(curr, lastDir)
	if err != nil {
		return nil, err
	}
	return dirObj, nil
}	


func NavigateFile(cat VirtualFS, path string) (VirtualFS, error) {
	dirs, file := DecomposePathFile(path)
	var curr VirtualFS = cat
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

func CreateFile(cat VirtualFS, name string, uuid string, metadata string) (VirtualFS, error) {
	_, found := cat.GetContent(name)
	if found {
		return nil, fmt.Errorf("entry %s already exists at %s", name, cat.Path())
	}
	now := time.Now()
	fileObj := &vfs_file{name, uuid, cat, now, now, metadata, -2}
	cat.SetContent(name, fileObj)
	if err := createCatalogFile(fileObj); err != nil {
		cat.DelContent(name)
		return nil, err
	}
	return fileObj, nil
}

func CreateDirectory(cat VirtualFS, name string) (VirtualFS, error) {
	_, found := cat.GetContent(name)
	if found {
		return nil, fmt.Errorf("entry %s already exists at %s", name, cat.Path())
	}
	dirObj := &vfs_dir{name, make(map[string]VirtualFS), cat, -2}
	cat.SetContent(name, dirObj)
	if err := createCatalogDirectory(dirObj); err != nil {
		cat.DelContent(name)
		return nil, err
	}
	return dirObj, nil
}
