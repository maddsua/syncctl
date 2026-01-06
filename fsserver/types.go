package fsserver

import (
	"fmt"
	"io"
	"time"
)

type ReadableFile struct {
	FileMetadata
	io.ReadSeekCloser
}

type FileUpload struct {
	FileMetadata
	io.Reader
}

type FileMetadata struct {
	Name   string
	Date   time.Time
	Size   int64
	SHA256 string
}

// todo: remove
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
