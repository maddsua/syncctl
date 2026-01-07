package storage_service

import (
	"context"
	"io"
	"time"
)

type BaseStorageController interface {
	Put(ctx context.Context, entry *FileUpload, overwrite bool) (*FileMetadata, error)
	Stat(ctx context.Context, name string) (*FileMetadata, error)
	Move(ctx context.Context, name string, newName string, overwrite bool) (*FileMetadata, error)
	Delete(ctx context.Context, name string) (*FileMetadata, error)
	List(ctx context.Context, prefix string, recursive bool, offset int, limit int) ([]FileMetadata, error)
}

type Storage interface {
	BaseStorageController
	Get(ctx context.Context, name string) (*ReadSeekableFile, error)
}

type StorageClient interface {
	BaseStorageController
	Download(ctx context.Context, name string) (*ReadableFile, error)
	Ping(ctx context.Context) error
}

type ReadSeekableFile struct {
	FileMetadata
	io.ReadSeekCloser
}

type ReadableFile struct {
	FileMetadata
	io.ReadCloser
	Offset int64
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
