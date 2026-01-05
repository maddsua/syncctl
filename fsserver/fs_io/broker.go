package fs_io

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
var HashFileExt = ".uploadhash"

func IsReservedExtension(val string) bool {
	switch path.Ext(val) {
	case PartFileExt, HashFileExt:
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

	var filter = func(nextPath string) error {

		if IsReservedExtension(nextPath) {
			return nil
		}

		scopedPath, _ := strings.CutPrefix(nextPath, broker.RootDir)
		if fileExt != "" && !strings.HasSuffix(scopedPath, fileExt) {
			return nil
		}

		stat, err := os.Stat(nextPath)
		if err != nil {
			return fmt.Errorf("stat file '%s': %v", scopedPath, err)
		}

		mtime := stat.ModTime()

		//	filter by creation time
		if !after.IsZero() && mtime.Before(after) {
			return nil
		} else if !before.IsZero() && mtime.After(before) {
			return nil
		}

		page.Total++

		//	apply pagination
		if offset > 0 && page.Total <= offset {
			return nil
		} else if limit > 0 && page.Size >= limit {
			return nil
		}

		hash, err := hashFileSHA256(nextPath, mtime)
		if err != nil {
			return fmt.Errorf("hash file '%s': %v", scopedPath, err)
		}

		page.Entries = append(page.Entries, fsserver.FileMetaEntry{
			Name:   scopedPath,
			Date:   mtime,
			Size:   stat.Size(),
			SHA256: hash,
		})

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
	} else if IsReservedExtension(entry.Name) {
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

	if err := os.Chtimes(tempPath, entry.Date, entry.Date); err != nil {
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

	//	todo: hash

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

//	todo: slightly refactor

func hashFileSHA256(baseFile string, baseFileMtime time.Time) (string, error) {

	cachePath := baseFile + HashFileExt

	var readCached = func() (string, error) {

		stat, err := os.Stat(cachePath)
		if err != nil || stat.ModTime() != baseFileMtime {
			return "", nil
		}

		file, err := os.Open(cachePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}

		hash, err := hex.DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			return "", err
		}

		return hex.EncodeToString(hash), nil
	}

	var storeCached = func(val string) (string, error) {

		var writeFile = func() error {

			file, err := os.Create(cachePath)
			if err != nil {
				return nil
			}

			defer file.Close()

			_, err = file.WriteString(val)
			return err
		}

		if err := writeFile(); err != nil {
			return "", err
		}

		if err := os.Chtimes(cachePath, baseFileMtime, baseFileMtime); err != nil {
			return "", err
		}

		return val, nil
	}

	var hashFile = func() (string, error) {

		file, err := os.Open(baseFile)
		if err != nil {
			return "", err
		}

		defer file.Close()

		hasher := sha256.New()

		if _, err := io.Copy(hasher, file); err != nil {
			return "", err
		}

		return hex.EncodeToString(hasher.Sum(nil)), nil
	}

	if val, _ := readCached(); val != "" {
		return val, nil
	}

	hash, err := hashFile()
	if err != nil {
		return "", err
	}

	return storeCached(hash)
}
