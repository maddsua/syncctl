package blobstorage

import (
	"archive/tar"
	"fmt"
	"io/fs"

	"github.com/maddsua/syncctl/fsserver"
)

func writeTarBlob(name string, entry *fsserver.FileUpload) error {
	//	todo: implement temp file write
}

func writeTarBlobMetaEntry(wrt *tar.Writer, key string, data []byte) error {

	if err := wrt.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     key,
		Size:     int64(len(data)),
		Mode:     int64(fs.ModePerm),
		Format:   tar.FormatGNU,
	}); err != nil {
		return err
	}

	if n, err := wrt.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("unexpected write size: %d", n)
	}

	return nil
}
