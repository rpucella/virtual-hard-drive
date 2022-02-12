
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
	Update() error       // Record changes to filesystem 
	AsVirtualFS() VirtualFS
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
	LocalPath() string   // This is the local path of the entry.
	FullPath() string    // Full path including drive name.    
	Parent() VirtualFS
	Root() VirtualFS
	Drive() Drive        // Returns the drive of this node (if any).
	Print()
	ContentList() []string
	GetContent(string) (VirtualFS, bool)
	SetContent(string, VirtualFS)
}

func walkCreateDirectories(cat VirtualFS, path string, directories []string) (VirtualFS, error) {
	// We can probably merge this function with NavigateCreateLast() below
	var curr VirtualFS
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
				dirObj = &vfs_dir{dir, currPath, make(map[string]VirtualFS), curr}
				curr.SetContent(dir, dirObj)
				curr = dirObj
			}
		} else {
			return nil, fmt.Errorf("unknown catalog object %v", curr)
		}
	}
	return curr, nil
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
	// It better not exist.
	if _, found := curr.GetContent(lastDir); found {
		return nil, fmt.Errorf("folder already exists: %s", lastDir)
	}
	dirObj := &vfs_dir{lastDir, curr.LocalPath() + lastDir + "/", make(map[string]VirtualFS), curr}
	curr.SetContent(lastDir, dirObj)
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

func AddFile(cat VirtualFS, name string, uuid string, metadata string) error {
	_, found := cat.GetContent(name)
	if found {
		return fmt.Errorf("file %s already exists at %s", name, cat.FullPath())
	}
	path := cat.LocalPath() + name
	now := time.Now()
	file := &vfs_file{name, path, uuid, cat, now, now, metadata}
	cat.SetContent(name, file)
	return nil
}
