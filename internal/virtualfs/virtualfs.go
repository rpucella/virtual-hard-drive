
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
	createFile(string, string, int, time.Time, time.Time, string) (int, error)
	createDirectory(string, int) (int, error)
	updateFile(int, string, int) error
	updateDirectory(int, string, int) error
	countFilesInDir(int) (int, error)
}

type File interface {
	Name() string
	UUID() string
	Created() time.Time
	Updated() time.Time
	Metadata() string       // Storage-specific metadata (such as # of chunks),
}

type Directory interface {
	Name() string
	CountFiles() (int, error)
}

type VirtualFS interface {
	IsFile() bool
	IsDir() bool
	IsRoot() bool
	IsDrive() bool
	AsDrive() Drive
	AsFile() File
	AsDir() Directory
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
	Move(VirtualFS, string) error
	CountFiles() (int, error)
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

func ValidateName(name string) error {
	if name == "." {
		return fmt.Errorf("name . not allowed")
	}
	if name == ".." {
		return fmt.Errorf("name .. not allowed")
	}
	return nil
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

func decomposePath(path string) []string {
	return strings.Split(path, "/")
}

func navigate(cat VirtualFS, path string, forceFile bool, forceDir bool, checkExists bool) (VirtualFS, error) {
	// Core function to navigate the virtual file system.
	// Use NavigatePath, NavigateDirectory, NavigateFile, NavigateParent as API.
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
		if forceFile {
			return nil, fmt.Errorf("file path ends with /: %s", path)
		}
		forceDir = true
	}
	dirs := decomposePath(cleanPath)
	if len(dirs) == 0 {
		return nil, fmt.Errorf("empty path to navigate")
	}
	var curr VirtualFS = cat
	for i, dir := range dirs {
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
			if !curr.IsDir() {
				return nil, fmt.Errorf("not a folder: %s", curr.Name())
			}
			newCurr, found := curr.GetContent(dir)
			if !found {
				if checkExists && (i == len(dirs) - 1) {
					// We're at the last name and we're checking for existence only.
					return nil, nil
				}
				return nil, fmt.Errorf("cannot find %s in %s", dir, curr.Path())
			}
			curr = newCurr
		}
	}
	if forceFile && !curr.IsFile() {
		return nil, fmt.Errorf("not a file: %s", path)
	}
	if forceDir && curr.IsFile() {
		return nil, fmt.Errorf("is a file: %s", path)
	}
	return curr, nil
}

func NavigatePath(cat VirtualFS, path string) (VirtualFS, error) {
	return navigate(cat, path, false, false, false)
}

func NavigateDirectory(cat VirtualFS, path string) (VirtualFS, error) {
	return navigate(cat, path, false, true, false)
}

func NavigateFile(cat VirtualFS, path string) (VirtualFS, error) {
	return navigate(cat, path, true, false, false)
}

func NavigateParent(cat VirtualFS, path string) (VirtualFS, string, error) {
	// Navigate to the parent of the path, returning that node and the final path
	// element.
	// This strips any trailing "/", so only really useful to implement mkdir.
	cleanPath := path
	if strings.HasSuffix(path, "/") {
		cleanPath = path[:len(path) - 1]
	}
	dirs := decomposePath("./" + cleanPath)  // Make sure we always have at least 2 entries in the path.
	if len(dirs) < 2 {
		return nil, "", fmt.Errorf("empty path to navigate")
	}
	parent, err := NavigateDirectory(cat, strings.Join(dirs[:len(dirs) - 1], "/"))
	if err != nil {
		return nil, "", err
	}
	return parent, dirs[len(dirs) - 1], nil
}

func CheckPath(cat VirtualFS, path string) (VirtualFS, error) {
	obj, err := navigate(cat, path, false, false, true)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
	
func CreateFile(dir VirtualFS, name string, uuid string, metadata string) (VirtualFS, error) {
	if dir.IsRoot() {
		return nil, fmt.Errorf("cannot create file in root")
	}
	_, found := dir.GetContent(name)
	if found {
		return nil, fmt.Errorf("entry %s already exists at %s", name, dir.Path())
	}
	now := time.Now()
	dirId := dir.CatalogId()
	if dir.IsDrive() {
		// Override if we're putting it in a drive
		dirId = -1
	}
	drive := dir.Drive()
	fileId, err := drive.createFile(name, uuid, dirId, now, now, metadata)
	if err != nil {
		return nil, err
	}
	fileObj := &vfs_file{name, uuid, dir, now, now, metadata, fileId}
	dir.SetContent(name, fileObj)
	return fileObj, nil
}

func CreateDirectory(dir VirtualFS, name string) (VirtualFS, error) {
	if dir.IsRoot() {
		return nil, fmt.Errorf("cannot create directory in root")
	}
	_, found := dir.GetContent(name)
	if found {
		return nil, fmt.Errorf("entry %s already exists at %s", name, dir.Path())
	}
	parentId := dir.CatalogId()
	if dir.IsDrive() {
		// Override if we're putting it in a drive
		parentId = -1
	}
	drive := dir.Drive()
	dirId, err := drive.createDirectory(name, parentId)
	if err != nil {
		return nil, err
	}
	dirObj := &vfs_dir{name, make(map[string]VirtualFS), dir, dirId}
	dir.SetContent(name, dirObj)
	return dirObj, nil
}

