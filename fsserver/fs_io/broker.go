package fs_io

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maddsua/syncctl/fsserver"
)

var ErrClosed = errors.New("broker closed")
var ErrNoFile = errors.New("file doesn't exist")
var ErrFileExists = errors.New("file already exists")

var PartFileExt = ".uploadpart"

func IsForbiddenExtension(val string) bool {
	switch path.Ext(val) {
	case PartFileExt:
		return true
	default:
		return false
	}
}

type FsBroker struct {
	RootDir string
	wg      sync.WaitGroup
	lock    sync.Mutex
	done    atomic.Bool
}

//	todo: clean up paths

func (broker *FsBroker) List(ctx context.Context, pathPrefix, fileExt string, after, before time.Time, offset, limit int) (*fsserver.Page[fsserver.FileMetaEntry], error) {

	if broker.done.Load() {
		return nil, ErrClosed
	}

	broker.wg.Add(1)
	defer broker.wg.Done()

	broker.lock.Lock()
	defer broker.lock.Unlock()

	if fileExt != "" && fileExt[0] != '.' {
		fileExt = "." + fileExt
	}

	page := fsserver.Page[fsserver.FileMetaEntry]{
		Offset: max(0, offset),
	}

	listPath := path.Join(broker.RootDir, pathPrefix)

	var filter = func(name string) error {

		scopedPath, _ := strings.CutPrefix(name, broker.RootDir)
		if fileExt != "" && !strings.HasSuffix(scopedPath, fileExt) {
			return nil
		}

		stat, err := os.Stat(name)
		if err != nil {
			return err
		}

		entry := fsserver.FileMetaEntry{
			Name: scopedPath,
			Date: stat.ModTime(),
			Size: stat.Size(),
		}

		//	filter by creation time
		if !after.IsZero() && entry.Date.Before(after) {
			return nil
		} else if !before.IsZero() && entry.Date.After(before) {
			return nil
		}

		page.Total++

		//	apply pagination
		if offset > 0 && page.Total <= offset {
			return nil
		} else if limit > 0 && page.Size >= limit {
			return nil
		}

		page.Entries = append(page.Entries, entry)

		page.Size++

		return nil
	}

	if err := iterateDir(ctx, filter, listPath); err != nil {
		return nil, err
	}

	return &page, nil
}

func (broker *FsBroker) Put(ctx context.Context, entry *fsserver.FileUpload, overwrite bool) (*fsserver.FileMetaEntry, error) {

	if entry.Name == "" {
		return nil, fmt.Errorf("empty file name")
	} else if IsForbiddenExtension(entry.Name) {
		return nil, fmt.Errorf("forbidden file extension (reserved)")
	}

	if broker.done.Load() {
		return nil, ErrClosed
	}

	broker.wg.Add(1)
	defer broker.wg.Done()

	distPath := path.Join(broker.RootDir, cleanNestedPath(entry.Name))

	if !overwrite {
		if _, err := os.Stat(distPath); err == nil {
			return nil, ErrFileExists
		}
	}

	if err := mkParentPath(distPath); err != nil {
		return nil, err
	}

	var writeFile = func(dest string) error {

		file, err := os.Create(dest)
		if err != nil {
			return err
		}

		defer file.Close()

		if err := os.Chtimes(dest, entry.Date, entry.Date); err != nil {
			return err
		}

		//	short-circuit to avoid unnecessary io calls
		if entry.Size == 0 || entry.Reader == nil {
			return nil
		}

		if n, err := io.Copy(file, entry.Reader); err != nil {
			return err
		} else if n != entry.Size {
			return fmt.Errorf("unexpected blob size %d instead of %d", n, entry.Size)
		}

		return nil
	}

	tempPath := path.Join(broker.RootDir, entry.Name+PartFileExt)
	if err := writeFile(tempPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}

	if err := os.Rename(tempPath, distPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}

	return &fsserver.FileMetaEntry{
		Name: cleanNestedPath(entry.Name),
		Date: entry.Date,
		Size: entry.Size,
	}, nil
}

func (broker *FsBroker) Get(ctx context.Context, name string) (*fsserver.File, error) {

	if broker.done.Load() {
		return nil, ErrClosed
	}

	//	automatic wg controls
	broker.wg.Add(1)
	defer broker.wg.Done()

	fsPath := path.Join(broker.RootDir, cleanNestedPath(name))

	stat, err := os.Stat(fsPath)
	if err != nil || !stat.Mode().IsRegular() {
		return nil, ErrNoFile
	}

	file, err := os.Open(fsPath)
	if err != nil {
		return nil, err
	}

	//	manually add one more to make sure we will wait until all operations are complete
	broker.wg.Add(1)

	return &fsserver.File{
		FileMetaEntry: fsserver.FileMetaEntry{
			Name: cleanNestedPath(name),
			Date: stat.ModTime(),
			Size: stat.Size(),
		},
		ReadSeekCloser: &fileReader{
			File:      file,
			WaitGroup: &broker.wg,
		},
	}, nil
}

func (broker *FsBroker) Move(ctx context.Context, oldPath, newPath string, overwrite bool) (*fsserver.FileMetaEntry, error) {

	if broker.done.Load() {
		return nil, ErrClosed
	}

	broker.wg.Add(1)
	defer broker.wg.Done()

	broker.lock.Lock()
	defer broker.lock.Unlock()

	src := path.Join(broker.RootDir, cleanNestedPath(oldPath))

	stat, err := os.Stat(src)
	if err != nil {
		return nil, ErrNoFile
	}

	entry := fsserver.FileMetaEntry{
		Name: cleanNestedPath(newPath),
		Date: stat.ModTime(),
		Size: stat.Size(),
	}

	dst := path.Join(broker.RootDir, cleanNestedPath(newPath))
	if !overwrite {
		if _, err := os.Stat(dst); err != nil {
			return nil, ErrFileExists
		}
	}

	if err := mkParentPath(dst); err != nil {
		return nil, err
	}

	if err := os.Rename(src, dst); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (broker *FsBroker) Delete(ctx context.Context, name string) error {

	if broker.done.Load() {
		return ErrClosed
	}

	broker.wg.Add(1)
	defer broker.wg.Done()

	broker.lock.Lock()
	defer broker.lock.Unlock()

	fsPath := path.Join(broker.RootDir, cleanNestedPath(name))

	if _, err := os.Stat(fsPath); err != nil {
		return ErrNoFile
	}

	if err := os.Remove(fsPath); err != nil {
		return err
	}

	return nil
}

type fileReader struct {
	*os.File
	*sync.WaitGroup
	done atomic.Bool
}

func (reader *fileReader) Close() error {
	if reader.done.CompareAndSwap(false, true) {
		reader.WaitGroup.Done()
	}
	return reader.File.Close()
}

func iterateDir(ctx context.Context, onFile func(name string) error, root string) error {

	if err := ctx.Err(); err != nil {
		return err
	}

	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {

		next := path.Join(root, entry.Name())

		if entry.Type().IsDir() {
			if err := iterateDir(ctx, onFile, next); err != nil {
				return err
			}
		} else if entry.Type().IsRegular() {
			if err := onFile(next); err != nil {
				return err
			}
		}
	}

	return nil
}

func cleanNestedPath(val string) string {
	const separator = "/"
	return path.Clean(separator + strings.TrimRight(val, separator))
}

func mkParentPath(val string) error {

	dir, _ := path.Split(val)
	if dir == "" {
		return nil
	}

	return os.MkdirAll(dir, os.ModePerm)
}
