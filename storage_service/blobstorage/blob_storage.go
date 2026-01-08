package blobstorage

import (
	"archive/tar"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	s4 "github.com/maddsua/syncctl/storage_service"
)

func CleanRelativePath(val string) string {
	const separator = "/"
	return path.Clean(separator + strings.TrimRight(val, separator))
}

func BlobPath(root, name string) string {
	return path.Join(root, CleanRelativePath(name)+FileExtBlob)
}

func TempBlobPath(root, name string) string {
	return path.Join(root, CleanRelativePath(name)+".*"+FileExtPartial)
}

func StripBlobPath(name, root string) string {
	return path.Clean(strings.TrimSuffix(strings.TrimPrefix(name, root), FileExtBlob))
}

func WalkDir(dir string, recursive bool, onFile func(name string) (wantMore bool, err error)) error {

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := path.Join(dir, entry.Name())
		if entry.IsDir() && recursive {
			if err := WalkDir(name, recursive, onFile); err != nil {
				return err
			}
		} else if entry.Type().IsRegular() && path.Ext(name) == FileExtBlob {
			if wantMore, err := onFile(name); err != nil {
				return err
			} else if !wantMore {
				break
			}
		}
	}

	return nil
}

type Storage struct {
	RootDir    string
	listLock   sync.Mutex
	uploadLock sync.Map
}

func (storage *Storage) Put(ctx context.Context, entry *s4.FileUpload, overwrite bool) (*s4.FileMetadata, error) {

	if entry.Name = CleanRelativePath(entry.Name); entry.Name == "" {
		return nil, &s4.NameError{Name: entry.Name}
	}

	if _, locked := storage.uploadLock.LoadOrStore(entry.Name, ctx); locked {
		return nil, &s4.FileConflictError{Path: entry.Name}
	}

	defer storage.uploadLock.Delete(entry.Name)

	blobPath := BlobPath(storage.RootDir, entry.Name)
	if _, err := os.Stat(blobPath); err == nil && !overwrite {
		return nil, &s4.FileConflictError{Path: entry.Name}
	}

	if err := os.MkdirAll(path.Dir(blobPath), fs.ModePerm); err != nil {
		return nil, err
	}

	tempBlob, err := WriteUploadAsBlob(TempBlobPath(storage.RootDir, entry.Name), entry)
	if err != nil {
		return nil, err
	}

	if err := os.Rename(tempBlob.Name, blobPath); err != nil {
		_ = os.Remove(tempBlob.Name)
		return nil, err
	}

	entry.FileMetadata.SHA256 = tempBlob.SHA256

	return &entry.FileMetadata, nil
}

func (storage *Storage) Get(ctx context.Context, name string) (*s4.ReadSeekableFile, error) {

	blobPath := BlobPath(storage.RootDir, name)

	stat, err := os.Stat(blobPath)
	if err != nil || !stat.Mode().IsRegular() {
		return nil, &s4.FileNotFoundError{Path: name}
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return nil, err
	}

	info, err := ReadBlobInfo(ctx, tar.NewReader(file))
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("read blob info: %v", err)
	}

	return &s4.ReadSeekableFile{
		FileMetadata: s4.FileMetadata{
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

func (storage *Storage) Stat(ctx context.Context, name string) (*s4.FileMetadata, error) {

	blobPath := BlobPath(storage.RootDir, name)

	if _, err := os.Stat(blobPath); err != nil {
		return nil, &s4.FileNotFoundError{Path: name}
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	info, err := ReadBlobInfo(ctx, tar.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("read blob info: %v", err)
	}

	return &s4.FileMetadata{
		Name:     CleanRelativePath(name),
		Size:     info.Size,
		Modified: info.Modified,
		SHA256:   info.SHA256,
	}, nil
}

func (storage *Storage) Move(ctx context.Context, name, newName string, overwrite bool) (*s4.FileMetadata, error) {

	storage.listLock.Lock()
	defer storage.listLock.Unlock()

	if name == "" {
		return nil, &s4.NameError{Name: name}
	} else if newName == "" {
		return nil, &s4.NameError{Name: newName}
	}

	stat, err := storage.Stat(ctx, name)
	if err != nil {
		return nil, err
	}

	blobPath := BlobPath(storage.RootDir, name)
	newBlobPath := BlobPath(storage.RootDir, newName)
	if _, err := os.Stat(newBlobPath); err == nil && !overwrite {
		return nil, &s4.FileConflictError{Path: name}
	}

	if err := os.MkdirAll(path.Dir(newBlobPath), os.ModePerm); err != nil {
		return nil, err
	}

	if err := os.Rename(blobPath, newBlobPath); err != nil {
		return nil, err
	}

	stat.Name = CleanRelativePath(newName)

	return stat, nil
}

func (storage *Storage) Delete(ctx context.Context, name string) (*s4.FileMetadata, error) {

	storage.listLock.Lock()
	defer storage.listLock.Unlock()

	stat, err := storage.Stat(ctx, name)
	if err != nil {
		return nil, err
	}

	blobPath := BlobPath(storage.RootDir, name)
	if err := os.Remove(blobPath); err != nil {
		return nil, err
	}

	return stat, nil
}

func (storage *Storage) List(ctx context.Context, prefix string, recursive bool, offset, limit int) ([]s4.FileMetadata, error) {

	storage.listLock.Lock()
	defer storage.listLock.Unlock()

	dirname := path.Dir(path.Join(storage.RootDir, prefix))

	stat, _ := os.Stat(dirname)
	if stat == nil || !stat.IsDir() {
		return nil, nil
	}

	var filterPrefix string
	if prefix != "" {
		filterPrefix = path.Join(storage.RootDir, prefix)
	}

	results := make([]s4.FileMetadata, 0)
	var pageIdx int

	var onFile = func(name string) (bool, error) {

		if filterPrefix != "" && !strings.HasPrefix(name, filterPrefix) {
			return true, nil
		}

		pageIdx++

		if offset > 0 && pageIdx <= offset {
			return true, nil
		} else if limit > 0 && len(results) >= limit {
			return false, nil
		}

		file, err := os.Open(name)
		if err != nil {
			return false, err
		}
		defer file.Close()

		info, err := ReadBlobInfo(ctx, tar.NewReader(file))
		if err != nil {
			return false, err
		}

		results = append(results, s4.FileMetadata{
			Name:     StripBlobPath(name, storage.RootDir),
			Size:     info.Size,
			Modified: info.Modified,
			SHA256:   info.SHA256,
		})

		return true, nil
	}

	if err := WalkDir(dirname, recursive, onFile); err != nil {
		return nil, err
	}

	return results, nil
}
