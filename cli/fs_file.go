package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"time"

	"github.com/maddsua/syncctl/utils"
)

type LocalFileInfo struct {
	Modified time.Time
	SHA256   string
}

func FileExists(name string) (*LocalFileInfo, error) {

	stat, _ := os.Stat(name)
	if stat == nil {
		return nil, nil
	}

	hasher := sha256.New()

	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	return &LocalFileInfo{
		Modified: stat.ModTime(),
		SHA256:   hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func WriteLocalFile(name string, reader io.Reader, mtime time.Time) error {

	var writeTmp = func() (*utils.FileJanitor, error) {

		dirname, basename := path.Split(name)

		file, err := os.CreateTemp(dirname, basename+"*.tmp")
		if err != nil {
			return nil, err
		}

		if _, err := io.Copy(file, reader); err != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
			return nil, err
		}

		if err := file.Close(); err != nil {
			return nil, err
		}

		return &utils.FileJanitor{Name: file.Name()}, nil
	}

	tmpFile, err := writeTmp()
	if err != nil {
		return err
	}
	defer tmpFile.Cleanup()

	if err := os.Chtimes(tmpFile.Name, mtime, mtime); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name, name); err != nil {
		return err
	}

	_ = tmpFile.Release()

	return nil
}
