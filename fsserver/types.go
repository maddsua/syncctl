package fsserver

import (
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
	Name     string
	Size     int64
	Modified time.Time
	SHA256   string
}

type StorageError struct {
	Message string
}

func (err *StorageError) Error() string {
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
