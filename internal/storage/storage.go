
package storage

type Storage interface {
	Name() string
	ListFiles() ([]string, error)
	DownloadFile(string, string, string) error
	UploadFile(string, string) (string, error)
	RemoteInfo(string, string) error
}

