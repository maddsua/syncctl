package fsserver

import (
	"fmt"
	"io"
	"time"
)

type File struct {
	FileMetaEntry
	io.ReadSeekCloser
}

type FileUpload struct {
	FileMetaEntry
	io.Reader
}

type FileMetaEntry struct {
	Name   string
	Date   time.Time
	Size   int64
	SHA256 string
}

type Page[T any] struct {
	Entries []T
	Size    int
	Offset  int
	Total   int
}

type FileFilterFn func(meta FileMetaEntry) bool

type Storage interface {
	Put(entry *FileUpload, overwrite bool) (*FileMetaEntry, error)
	Get(name string) (*File, error)
	Move(oldPath string, newPath string, overwrite bool) (*FileMetaEntry, error)
	Delete(name string) error
	Find(filter FileFilterFn) ([]FileMetaEntry, error)
}

type StorageError struct {
	Message string
	Cause   string
}

func (err *StorageError) Error() string {

	if err.Cause != "" {
		return fmt.Sprintf("%s: %s", err.Message, err.Cause)
	}

	return err.Message
}

var ErrNoFile = &StorageError{
	Message: "file not found",
}

var ErrFileConflict = &StorageError{
	Message: "file conflict",
}

var ErrInvalidFileName = &StorageError{
	Message: "invalid file name",
}
