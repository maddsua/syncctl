package blobstorage

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/maddsua/syncctl/fsserver"
)

func CleanRelativePath(val string) string {
	const separator = "/"
	return path.Clean(separator + strings.TrimRight(val, separator))
}

var ErrClosed = errors.New("storage closed")

const FileExtBlob = ".blob"
const FileExtPartial = ".part"

type Storage struct {
	RootDir string
	wg      sync.WaitGroup
	lock    sync.Mutex
	done    atomic.Bool
}

func (storage *Storage) Put(entry *fsserver.FileUpload, overwrite bool) (*fsserver.FileMetaEntry, error) {

	if storage.done.Load() {
		return nil, ErrClosed
	}

	storage.wg.Add(1)
	defer storage.wg.Done()

	if entry.Name = CleanRelativePath(entry.Name); entry.Name == "" {
		return nil, fsserver.ErrInvalidFileName
	}

	blobPath := path.Join(storage.RootDir, entry.Name+FileExtBlob)

	if _, err := os.Stat(blobPath); err == nil && !overwrite {
		return nil, fsserver.ErrFileConflict
	}

	if err := os.MkdirAll(path.Dir(blobPath), fs.ModePerm); err != nil {
		return nil, err
	}

	tempBlobPath := blobPath + FileExtPartial
	if err := WriteUploadAsBlob(tempBlobPath, entry); err != nil {
		return nil, err
	}

	if err := os.Rename(tempBlobPath, blobPath); err != nil {
		return nil, err
	}

	return &entry.FileMetaEntry, nil
}
