
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

func (s LocalFileSystem) ListFiles() ([]string, error) {
	result := make([]string, 0, 10)
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

func (s LocalFileSystem) WriteFile(content []byte, target string) error {
	path := path.Join(s.root, target)
	err := os.WriteFile(path, content, 0600)
	if err != nil {
		return fmt.Errorf("os.WriteFile: %v", err)
	}
	return nil
}

func (s LocalFileSystem) DownloadFile(uuid string, metadata string, outputFileName string) error {
	path := path.Join(s.root, uuid)
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

func (s LocalFileSystem) UploadFile(file string, target string) (string, error) {
	src, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("os.Open: %v", err)
	}
	defer src.Close()

	path := path.Join(s.root, target)
	dest, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("os.Create: %v", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}
	
	return "", nil
}

func (s LocalFileSystem) RemoteInfo(uuid string, metadata string) error {
	path := path.Join(s.root, uuid)
	attrs, err := os.Stat(path)
	if err != nil {
		fmt.Errorf("os.Stat: %v", err)
	}
	fmt.Printf("Remote:      %s\n", s.Name())
	if attrs.Size() < 1024 {
		fmt.Printf(" %s  %4d B\n", attrs.Name(), attrs.Size())
	} else if attrs.Size() < 1024 * 1024 {
		size := attrs.Size() / 1024
		fmt.Printf(" %s  %4d MiB\n", attrs.Name(), size)
	} else {
		size := attrs.Size() / (1024 * 1024)
		fmt.Printf(" %s  %4d GiB\n", attrs.Name(), size)
	}
	return nil
}
