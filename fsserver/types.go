package fsserver

import (
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
