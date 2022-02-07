
package storage

type Storage interface {
	Name() string
	ListFiles() ([]string, error)
	ReadFile(string) ([]byte, error)
	DownloadFile(string, string) error
}

