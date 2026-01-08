package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/maddsua/syncctl/utils"
)

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
