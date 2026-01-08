package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

type FileJanitor struct {
	Name string

	//	A flag to tell this cleanup thingy to fuck off.
	//	Not using atomic values here since it's not intended for concurrent execution,
	// 	but rather to avoid variable fuckery inside function body
	released bool
}

func (janitor *FileJanitor) Release() string {
	janitor.released = true
	return janitor.Name
}

func (janitor *FileJanitor) Cleanup() error {
	if !janitor.released {
		return os.Remove(janitor.Name)
	}
	return nil
}

func NamedFileHashSha256(name string) (string, error) {

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

func FileHashSha256(file *os.File) (string, error) {

	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func WriteTempFile(dirname, basename string, reader io.Reader) (*FileJanitor, error) {

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

	return &FileJanitor{Name: file.Name()}, nil
}
