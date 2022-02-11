
package virtualfs

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"
	"path"
	"time"
	"strconv"
	
        "gopkg.in/yaml.v3"
	
	"rpucella.net/virtual-hard-drive/internal/storage"
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_FILE = "config.yml"
const CONFIG_CATALOG = "catalog"

type config struct {
	Type string
	Location string
	Description string
}

type drive struct{
	name string
	description string
	catalogPath string    // This could be kept private.
	storage storage.Storage
	top VirtualFS          // This is a horrible name.
	root VirtualFS
	// Add possible restriction flags (i.e., warn in case of too recent deletes, etc)
}

func (d *drive) Name() string {
	return d.name
}

func (d *drive) Description() string {
	return d.description
}

func (d *drive) Storage() storage.Storage {
	return d.storage
}

func (d *drive) AsVirtualFS() VirtualFS {
	return d
}

func (d *drive) AsDrive() Drive {
	return d
}

func (r *drive) IsFile() bool {
	return false
}

func (r *drive) AsFile() File {
	return nil
}

func (r *drive) IsDir() bool {
	return true
}

func (r *drive) LocalPath() string {
	return fmt.Sprintf("/%s/", r.name)
}

func (r *drive) FullPath() string {
	return fmt.Sprintf("/%s/", r.name)
}

func (r *drive) Parent() VirtualFS {
	return r.root
}

func (r *drive) Root() VirtualFS {
	return r.root
}

func (r *drive) Drive() Drive {
	return r
}

func (r *drive) Print() {
	fmt.Printf("<Drive %s>\n", r.name)
}

func (r *drive) ContentList() []string {
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	return r.top.ContentList()
}

func (r *drive) GetContent(field string) (VirtualFS, bool) {
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	result, found := r.top.GetContent(field)
	return result, found
}

func (r *drive) SetContent(name string, value VirtualFS) {
	// Do nothing silently?
	if r.top == nil {
		if err := fetchCatalog(r); err != nil {
			fmt.Printf("ERROR THAT CANNOT BE CAUGHT when fetching catalog for %s\n%w\n", r.name, err)
		}
	}
	r.top.SetContent(name, value)
	// Call to UpdateVirtualFS?
}

// Store catalogs locally.

func fetchCatalog(dr *drive) error {
	flat, err := ioutil.ReadFile(dr.catalogPath)
	if err != nil {
		return fmt.Errorf("cannot fetch catalog: %w", err)
	}
	// Convert to a string first.
	strFlat := string(flat)
	///fmt.Printf("Flat: [%s]\n", strFlat)
	lines := strings.Split(strFlat, "\n")
	root := dr.root
	dr.top = &vfs_dir{"", "/", make(map[string]VirtualFS), root}
	cat := dr
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if len(line) > 0 {
			// Skip empty lines.
			path, uuid, updated, created, err := splitLine(line)
			if err != nil {
				return fmt.Errorf("cannot parse catalog: %w", err)
			}
			if uuid == "" {
				// Directory only.
				directories, err := splitPath(path)
				if err != nil {
					return fmt.Errorf("cannot parse catalog: %w", err)
				}
				if _, err := walkCreateDirectories(cat, path, directories); err != nil {
					return fmt.Errorf("cannot parse catalog: %w", err)
				}
			} else { 
				directories, file, err := splitPathFile(path)
				if err != nil {
					return fmt.Errorf("cannot parse catalog: %w", err)
				}
				curr, err := walkCreateDirectories(cat, path, directories)
				currPath := curr.LocalPath()
				// At this point, curr is in the directory where we want the file.
				if curr.IsFile() {
					return fmt.Errorf("file in middle of path %s", path)
				}
				_, exists := curr.GetContent(file)
				if exists {
					return fmt.Errorf("file %s already exists in path %s", file, path)
				}
				upTime := time.Unix(updated, 0)
				crTime := time.Unix(created, 0)
				fileObj := &vfs_file{file, currPath + file, uuid, curr, crTime, upTime}
				curr.SetContent(file, fileObj)
			}
		}
	}
	return nil
}

func (dr *drive) Update() error {
	flatCat := Flatten(dr.top)
	catFile := []byte(strings.Join(flatCat, "\n") + "\n")
	// Have we created a .tmp backup backup?
	made_tmp := false
	// Backup catalog.bak into catalog.tmp if it exists.
	if _, err := os.Stat(dr.catalogPath + ".bak"); err == nil {
		// Backup exists, so keep it.
		if err := os.Rename(dr.catalogPath + ".bak", dr.catalogPath + ".tmp"); err != nil {
			return fmt.Errorf("cannot temporarily preserve backup catalog")
		}
		made_tmp = true
	}
	// Backup catalog into catalog.bak.
	if err := os.Rename(dr.catalogPath, dr.catalogPath + ".bak"); err != nil {
		if made_tmp { 
			if err2 := os.Rename(dr.catalogPath + ".tmp", dr.catalogPath + ".bak"); err2 != nil {
				return fmt.Errorf("cannot create backup catalog (%w) or restore tmp backup (%w)", err, err2)
			}
		}
		return fmt.Errorf("cannot create backup catalog: %w", err)
	}
	// Write catalog.
	err := ioutil.WriteFile(dr.catalogPath, catFile, 0600)
	if err != nil {
		if err2 := os.Rename(dr.catalogPath + ".bak", dr.catalogPath); err2 != nil {
			return fmt.Errorf("cannot update catalog (%w) or restore backup (%w)", err, err2)
		}
		if made_tmp {
			if err2 := os.Rename(dr.catalogPath + ".tmp", dr.catalogPath + ".bak"); err2 != nil {
				return fmt.Errorf("cannot update catalog (%w) or restore tmp backup (%w)", err, err2)
			}
		}
		return fmt.Errorf("cannot update catalog: %s", err)
	}
	// Remove catalog.tmp since no longer needed.
	if made_tmp {
		if err := os.Remove(dr.catalogPath + ".tmp"); err != nil {
			return fmt.Errorf("cannot remote tmp backup: %w", err)
		}
	}
	return nil
}

func readDrives(root Root) (map[string]Drive, error) {
	home, err := os.UserHomeDir()
	if err != nil { 
		return nil, fmt.Errorf("cannot get home directory: %v", err)
	}
	configFolder := path.Join(home, CONFIG_FOLDER)
	info, err := os.Stat(configFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(configFolder, 0700)
		if err != nil {
			return nil, fmt.Errorf("cannot create %s directory: %w", configFolder, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot access %s directory: %w", configFolder, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path %s not a directory", configFolder)
	}
	files, err := ioutil.ReadDir(configFolder)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s directory: %w", configFolder, err)
	}
	drives:= make(map[string]Drive)
	for _, f := range files {
		if f.IsDir() {
			yamlFile, err := ioutil.ReadFile(path.Join(configFolder, f.Name(), CONFIG_FILE))
			// Skip errors silently.
			if err == nil {
				config := config{}
				err := yaml.Unmarshal(yamlFile, &config)
				// Again, skip errors silently.
				if err == nil {
					var store storage.Storage
					if config.Type == "gcs" {
						store = storage.NewGoogleCloud(config.Location)
					} else if config.Type == "local" {
						store = storage.NewLocalFileSystem(config.Location)
					} else {
						// Unknown type - skip silently.
						continue
					}
					catalogPath := path.Join(configFolder, f.Name(), CONFIG_CATALOG)
					drives[f.Name()] = &drive{f.Name(), config.Description, catalogPath, store, nil, root.AsVirtualFS()}
				}
			}
		}
	}
	return drives, nil
}

func splitLine(line string) (string, string, int64, int64, error) {
	ss := strings.Split(line, ":")
	if len(ss) < 1 {
		// Allow for at least two fields - path and file UUID.
		// Other fields (date, etc) may be optional, with defaults.
		return "", "", 0, 0, fmt.Errorf("wrong number of fields in line %d", len(ss))
	}
	uuid := ""
	updated := int64(0)
	created := int64(0)
	if len(ss) > 1 {
		uuid = ss[1]
	}
	if len(ss) > 2 {
		newUpdated, err := strconv.ParseInt(ss[2], 10, 64)
		// In case of error, default to 0.
		if err == nil {
			updated = newUpdated
		}
	}
	if len(ss) > 3 {
		newCreated, err := strconv.ParseInt(ss[3], 10, 64)
		// In case of error, default to 0.
		if err == nil {
			created = newCreated
		}
	}
	return ss[0], uuid, updated, created, nil
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


func flatten(cat VirtualFS, prefix string) []string {
	result := make([]string, 0)     // cat.Size()
	if cat == nil { 
		return result
	}
	if file := cat.AsFile(); file != nil {
		line := fmt.Sprintf("%s:%s:%d:%d", prefix, file.UUID(), file.Updated().Unix(), file.Created().Unix())
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

func Flatten(cat VirtualFS) []string {
	return flatten(cat, "")
}
