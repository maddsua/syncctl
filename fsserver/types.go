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

// todo: replace
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
