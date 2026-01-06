package blobstorage

import (
	"archive/tar"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/maddsua/syncctl/fsserver"
)

var ErrClosed = errors.New("storage closed")

const FileExtBlob = ".blob"
const FileExtPartial = ".part"

func CleanRelativePath(val string) string {
	const separator = "/"
	return path.Clean(separator + strings.TrimRight(val, separator))
}

func BlobPath(root, name string) string {
	return path.Join(root, CleanRelativePath(name)+FileExtBlob)
}

func TempBlobPath(root, name string) string {
	return path.Join(root, CleanRelativePath(name)+FileExtBlob+FileExtPartial)
}

type Storage struct {
	RootDir string
	lock    sync.Mutex
}

func (storage *Storage) Put(entry *fsserver.FileUpload, overwrite bool) (*fsserver.FileMetadata, error) {

	if entry.Name = CleanRelativePath(entry.Name); entry.Name == "" {
		return nil, fsserver.ErrInvalidFileName
	}

	blobPath := BlobPath(storage.RootDir, entry.Name)

	if _, err := os.Stat(blobPath); err == nil && !overwrite {
		return nil, fsserver.ErrFileConflict
	}

	if err := os.MkdirAll(path.Dir(blobPath), fs.ModePerm); err != nil {
		return nil, err
	}

	tempBlobPath := TempBlobPath(storage.RootDir, entry.Name)
	if meta, err := WriteUploadAsBlob(tempBlobPath, entry); err != nil {
		return nil, err
	} else {
		entry.FileMetadata.SHA256 = meta.SHA256
	}

	if err := os.Rename(tempBlobPath, blobPath); err != nil {
		return nil, err
	}

	return &entry.FileMetadata, nil
}

func (storage *Storage) Get(name string) (*fsserver.ReadableFile, error) {

	blobPath := BlobPath(storage.RootDir, name)

	stat, err := os.Stat(blobPath)
	if err != nil || !stat.Mode().IsRegular() {
		return nil, fsserver.ErrNoFile
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return nil, err
	}

	info, err := ReadBlobInfo(tar.NewReader(file))
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("read blob info: %v", err)
	}

	return &fsserver.ReadableFile{
		FileMetadata: fsserver.FileMetadata{
			Name:     CleanRelativePath(name),
			Modified: info.Modified,
			Size:     info.Size,
			SHA256:   info.SHA256,
		},
		ReadSeekCloser: &BlobReader{
			File: file,
		},
	}, nil
}
