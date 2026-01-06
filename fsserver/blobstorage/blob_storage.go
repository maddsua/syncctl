package blobstorage

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/maddsua/syncctl/fsserver"
)

var ErrClosed = errors.New("storage closed")

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

	if entry.Name == "" {
		return nil, fsserver.ErrInvalidFileName
	}

	//	todo: write tarball
}
