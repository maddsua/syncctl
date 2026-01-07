package blobstorage

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/utils"
)

const FileExtBlob = ".blob"
const FileExtPartial = ".part"

const blobKeyMetadata = "metadata"
const blobKeyData = "data"

type BlobInfo struct {
	BlobMetadata
	Size     int64
	Modified time.Time
}

type TempBlobInfo struct {
	Name string
	BlobInfo
}

func WriteUploadAsBlob(name string, entry *s4.FileUpload) (*TempBlobInfo, error) {

	file, err := os.CreateTemp(path.Split(name))
	if err != nil {
		return nil, &BlobError{"create partial file", err}
	}

	janitor := utils.FileJanitor{Name: file.Name()}

	defer janitor.Cleanup()
	defer file.Close()

	arc := tar.NewWriter(file)

	if err := arc.WriteHeader(&tar.Header{
		Format:     tar.FormatGNU,
		Typeflag:   tar.TypeReg,
		Name:       blobKeyData,
		Size:       entry.Size,
		Mode:       int64(os.FileMode(0660).Perm()),
		ModTime:    entry.Modified,
		AccessTime: entry.Modified,
		ChangeTime: entry.Modified,
	}); err != nil {
		return nil, &BlobError{"write tar data entry header", err}
	}

	hasher := sha256.New()

	if n, err := io.Copy(arc, io.TeeReader(entry.Reader, hasher)); err != nil {
		return nil, &BlobError{"write tar data entry", err}
	} else if n != entry.Size {
		return nil, &BlobError{"write tar data entry", fmt.Errorf("expected size: %d bytes but wrote %d instead", n, entry.Size)}
	}

	meta := BlobMetadata{
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}

	if entry.SHA256 != "" {
		if meta.SHA256 != entry.SHA256 {
			return nil, &BlobError{"data entry sha256 checksum", fmt.Errorf("expected: '%s'; have '%s'", meta.SHA256, entry.SHA256)}
		}
	}

	if err := meta.WriteTar(arc); err != nil {
		return nil, err
	}

	if err := arc.Close(); err != nil {
		return nil, err
	}

	//	calling it one more time just to make sure we don't get any wonky writes
	if err := file.Close(); err != nil {
		return nil, err
	}

	return &TempBlobInfo{
		Name: janitor.Release(),
		BlobInfo: BlobInfo{
			BlobMetadata: meta,
			Size:         entry.Size,
			Modified:     entry.Modified,
		},
	}, nil
}

func ReadBlobInfo(ctx context.Context, reader *tar.Reader) (*BlobInfo, error) {

	var info BlobInfo

	readSet := map[string]struct{}{}

	for {

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		entry, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, &BlobError{"read next tar entry", err}
		}

		switch entry.Name {
		case blobKeyData:
			info = BlobInfo{
				Size:     entry.Size,
				Modified: entry.ModTime,
			}
			readSet[blobKeyData] = struct{}{}
		case blobKeyMetadata:
			if err := info.BlobMetadata.ReadTar(reader); err != nil {
				return nil, &BlobError{"read tar metadata entry", err}
			}
			readSet[blobKeyMetadata] = struct{}{}
		}
	}

	if _, has := readSet[blobKeyData]; !has {
		return nil, &BlobError{"format check", fmt.Errorf("missing data entry")}
	} else if _, has := readSet[blobKeyMetadata]; !has {
		return nil, &BlobError{"format check", fmt.Errorf("missing metadata entry found")}
	}

	return &info, nil
}
