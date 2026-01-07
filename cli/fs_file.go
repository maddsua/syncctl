package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/maddsua/syncctl/utils"
)

type FileContentStats struct {
	Modified time.Time
	SHA256   string
}

func FileContentStat(name string) (*FileContentStats, error) {

	stat, _ := os.Stat(name)
	if stat == nil {
		return nil, nil
	}

	hash, err := FileSha256HashString(name)
	if err != nil {
		return nil, err
	}

	return &FileContentStats{
		Modified: stat.ModTime(),
		SHA256:   hash,
	}, nil
}

func FileSha256HashString(name string) (string, error) {

	hasher := sha256.New()

	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func WriteTempFile(dirname, basename string, reader io.Reader) (*utils.FileJanitor, error) {

	file, err := os.CreateTemp(dirname, basename+"*.tmp")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}

	//	not deferring because we kinda want to check if it's succeeded
	if err := file.Close(); err != nil {
		return nil, err
	}

	return &utils.FileJanitor{Name: file.Name()}, nil
}
