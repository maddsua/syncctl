package blobstorage

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io/fs"
)

const blobKeyMetadata = "metadat"
const blobKeyData = "data"

type BlobMetadata struct {
	SHA256 string `json:"h"`
}

func (meta *BlobMetadata) WriteTar(wrt *tar.Writer) error {

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if err := wrt.WriteHeader(&tar.Header{
		Format:   tar.FormatGNU,
		Typeflag: tar.TypeReg,
		Name:     blobKeyMetadata,
		Size:     int64(len(data)),
		Mode:     int64(fs.ModePerm),
	}); err != nil {
		return fmt.Errorf("write header: %v", err)
	}

	if n, err := wrt.Write(data); err != nil {
		return fmt.Errorf("write content: %v", err)
	} else if n != len(data) {
		return fmt.Errorf("unexpected write size: %d", n)
	}

	return nil
}

func (meta *BlobMetadata) ReadTar(reader *tar.Reader) error {
	return json.NewDecoder(reader).Decode(meta)
}
