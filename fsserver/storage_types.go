package fsserver

import (
	"io"
	"time"
)

type Storage interface {
	Put(entry *FileUpload, overwrite bool) (*FileMetadata, error)
	Get(name string) (*ReadableFile, error)
	Stat(name string) (*FileMetadata, error)
	Move(name string, newName string, overwrite bool) (*FileMetadata, error)
	Delete(name string) (*FileMetadata, error)
	List(prefix string, recursive bool, offset int, limit int) ([]FileMetadata, error)
}

type ReadableFile struct {
	FileMetadata
	io.ReadSeekCloser
}

type FileUpload struct {
	FileMetadata
	io.Reader
}

type FileMetadata struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"mod"`
	SHA256   string    `json:"sha256"`
}
