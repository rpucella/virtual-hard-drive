
package catalog

import (
	"time"
)

// What's an abstract interface to a catalog?

type DriveDescriptor struct {
	Id int
	Name string
	Type string
	Location string
	Description string
}

type DirectoryDescriptor struct {
	Id int
	Name string
	ParentId int
}

type FileDescriptor struct {
	id int
	Name string
	DirectoryId int
	UUID string
	Created time.Time
	Updated time.Time
	Metadata string
}

type Catalog interface {
	FetchDrives() (map[int]DriveDescriptor, error)
	FetchFiles(int) (map[int]FileDescriptor, error)
	FetchDirectories(int) (map[int]DirectoryDescriptor, error)
	CreateFile(int, string, string, int, time.Time, time.Time, string) (int, error)
	CreateDirectory(int, string, int) (int, error)
	UpdateFile(int, string, int) error
	UpdateDirectory(int, string, int) error
}

