package blobstorage

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
)

type BlobReader struct {
	File   *os.File
	arc    *tar.Reader
	entry  *tar.Header
	offset int64
}

func (reader *BlobReader) seekBlobStart() error {

	if _, err := reader.File.Seek(0, io.SeekStart); err != nil {
		return err
	}

	reader.arc = tar.NewReader(reader.File)

	for {

		entry, err := reader.arc.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if entry.Name == blobKeyData {
			reader.entry = entry
			reader.offset = 0
			return nil
		}
	}

	return &BlobError{"blob init", errors.New("missing data entry")}
}

func (reader *BlobReader) initBlob() error {

	if reader.entry != nil && reader.entry.Name == blobKeyData {
		return nil
	}

	return reader.seekBlobStart()
}

func (reader *BlobReader) seek(newOffset int64) (int64, error) {

	if newOffset < 0 || newOffset >= reader.entry.Size {
		return -1, &BlobError{"seek", errors.New("invalid offset")}
	} else if newOffset == reader.offset {
		return reader.offset, nil
	}

	if newOffset < reader.offset {
		if err := reader.seekBlobStart(); err != nil {
			return -1, err
		}
	}

	n, err := io.CopyN(io.Discard, reader.arc, newOffset-reader.offset)
	if n > 0 {
		reader.offset += n
	}

	return reader.offset, err
}

func (reader *BlobReader) Read(buff []byte) (int, error) {

	if err := reader.initBlob(); err != nil {
		return -1, err
	}

	n, err := reader.arc.Read(buff)

	if n > 0 {
		reader.offset += int64(n)
	}

	return n, err
}

func (reader *BlobReader) Seek(offset int64, whence int) (int64, error) {

	if err := reader.initBlob(); err != nil {
		return -1, err
	}

	switch whence {
	case io.SeekCurrent:
		return reader.seek(reader.offset + offset)
	case io.SeekStart:
		return reader.seek(offset)
	case io.SeekEnd:
		return reader.seek(reader.entry.Size + offset)
	default:
		return -1, &BlobError{"seek", fmt.Errorf("invalid whence: %d", whence)}
	}
}

func (reader *BlobReader) Close() error {
	return reader.File.Close()
}
