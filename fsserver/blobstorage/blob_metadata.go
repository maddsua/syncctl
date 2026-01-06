package blobstorage

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io/fs"
	"time"
)

type BlobMetadata struct {
	SHA256 string `json:"h"`
}

func (meta *BlobMetadata) WriteTar(wrt *tar.Writer) error {

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if err := wrt.WriteHeader(&tar.Header{
		Format:     tar.FormatGNU,
		Typeflag:   tar.TypeReg,
		Name:       blobKeyMetadata,
		Size:       int64(len(data)),
		Mode:       int64(fs.ModePerm),
		ChangeTime: time.Now(),
		AccessTime: time.Now(),
		ModTime:    time.Now(),
	}); err != nil {
		return &BlobError{"write tar metadata entry header", err}
	}

	if n, err := wrt.Write(data); err != nil {
		return &BlobError{"write tar metadata entry", err}
	} else if n != len(data) {
		return &BlobError{"write tar metadata entry", fmt.Errorf("expected size: %d bytes but wrote %d instead", n, len(data))}
	}

	return nil
}

func (meta *BlobMetadata) ReadTar(reader *tar.Reader) error {
	return json.NewDecoder(reader).Decode(meta)
}
