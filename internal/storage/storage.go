
package storage

type Storage interface {
	Name() string
	UUIDToPath(string) (string, error)
	CatalogToPath(string) (string, error)
	ListFiles() ([]string, error)
	ReadFile(string) ([]byte, error)
	DownloadFile(string, string) error
}

