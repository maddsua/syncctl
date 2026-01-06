package blobstorage

import "fmt"

type BlobError struct {
	Operation string
	Err       error
}

func (err *BlobError) Error() string {
	return fmt.Sprintf("%s: %v", err.Operation, err.Err)
}
