package blobstorage

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/maddsua/syncctl/fsserver"
)

func WriteUploadAsBlob(name string, entry *fsserver.FileUpload) (*BlobMetadata, error) {

	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	arc := tar.NewWriter(file)

	if err := arc.WriteHeader(&tar.Header{
		Format:     tar.FormatGNU,
		Typeflag:   tar.TypeReg,
		Name:       blobKeyData,
		Size:       entry.Size,
		Mode:       int64(fs.ModePerm),
		ModTime:    entry.Date,
		AccessTime: entry.Date,
		ChangeTime: entry.Date,
	}); err != nil {
		return nil, fmt.Errorf("write data block header: %v", err)
	}

	hasher := sha256.New()

	if n, err := io.Copy(arc, io.TeeReader(entry.Reader, hasher)); err != nil {
		return nil, fmt.Errorf("write data block content: %v", err)
	} else if n != entry.Size {
		return nil, fmt.Errorf("unexpected blob size: %d bytes written instead of %d", n, entry.Size)
	}

	meta := BlobMetadata{
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}

	if entry.SHA256 != "" {
		if meta.SHA256 != entry.SHA256 {
			return nil, fmt.Errorf("unexpected sha256 checksum: '%s' instead of '%s'", meta.SHA256, entry.SHA256)
		}
	}

	if err := meta.WriteTar(arc); err != nil {
		return nil, fmt.Errorf("write metadata: %v", err)
	}

	return &meta, arc.Close()
}
