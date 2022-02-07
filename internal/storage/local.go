
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"path"
)

type LocalFileSystem struct {
	root string
}

func NewLocalFileSystem(root string) LocalFileSystem {
	return LocalFileSystem{root}
}

func (s LocalFileSystem) Name() string {
	return fmt.Sprintf("local::%s", s.root)
}

func (s LocalFileSystem) UUIDToPath(uuid string) (string, error) {
	if len(uuid) != 36 {
		return "", fmt.Errorf("length of UUID %s <> 36", uuid)
	}
	return uuid, nil
}

func (s LocalFileSystem) ListFiles() ([]string, error) {
	result := make([]string, 10)
	accumulate := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			result = append(result, path)
		}
		return nil
	}
	err := filepath.Walk(s.root, accumulate)
	if err != nil {
		return nil, fmt.Errorf("filepath.Walk: %v", err)
	}
	return result, nil
}

func (s LocalFileSystem) ReadFile(file string) ([]byte, error) {
	path := path.Join(s.root, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %v", err)
	}
	return data, nil
}

func (s LocalFileSystem) DownloadFile(file string, outputFileName string) error {
	path := path.Join(s.root, file)
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer src.Close()

	dest, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	return nil
}
