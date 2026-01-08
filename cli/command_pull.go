package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

//	todo: fix error handling

func Pull(ctx context.Context, client s4.StorageClient, remoteDir, localDir string, onconflict ConflictResolutionPolicy, prune bool) error {

	if onconflict == ResolveAsVersions {
		prune = false
	}

	pruneMap := map[string]struct{}{}

	if prune {

		fmt.Println("Indexing local files...")

		entries, err := ListAllRegularFiles(localDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			pruneMap[path.Clean(entry)] = struct{}{}
		}
	}

	fmt.Println("Fetching remote index...")

	remoteFiles, err := client.List(ctx, remoteDir, true, 0, 0)
	if err != nil {
		return err
	} else if len(remoteFiles) == 0 {
		fmt.Println("No files on the remote. Exiting.")
		return nil
	}

	for _, entry := range remoteFiles {

		localPath := path.Join(localDir, strings.TrimPrefix(path.Clean(entry.Name), path.Clean(remoteDir)))

		if err := pullEntry(ctx, client, localPath, onconflict, &entry); err != nil {
			fmt.Fprintf(os.Stderr, "--X Error pulling '%s':\n", entry.Name)
			fmt.Fprintf(os.Stderr, "    %v\n", err)
			return err
		}

		delete(pruneMap, localPath)
	}

	if prune {
		for name := range pruneMap {
			if err := os.Remove(name); err != nil {
				return err
			}
			fmt.Println("--> Prune", name)
		}
	}

	fmt.Println("Pull complete")

	return nil
}

func pullEntry(ctx context.Context, client s4.StorageClient, localPath string, onconflict ConflictResolutionPolicy, entry *s4.FileMetadata) error {

	if stat, err := os.Stat(localPath); err != nil {
		return err
	} else if stat == nil {
		fmt.Printf("--> Downloading '%s' (%s)\n", localPath, DataSizeString(float64(entry.Size)))
	} else {

		hash, err := NamedFileHashSha256(localPath)
		if err != nil {
			return err
		}

		switch onconflict {

		case ResolveOverwrite:

			if hash == entry.SHA256 {

				fmt.Printf("--> Up to date '%s'\n", localPath)

				if !stat.ModTime().Equal(entry.Modified) {
					fmt.Printf("    --> Update mtime '%s'\n", localPath)
					if err := os.Chtimes(localPath, entry.Modified, entry.Modified); err != nil {
						return err
					}
				}

				return nil
			}

			fmt.Printf("--> Updating '%s' (%s)\n", localPath, DataSizeString(float64(entry.Size)))

		case ResolveAsVersions:

			if hash == entry.SHA256 {
				fmt.Printf("--> Up to date '%s'\n", localPath)
				return nil
			}

			idx, err := HighestFileIndex(localPath)
			if err != nil {
				return err
			}

			if hash, err := NamedFileHashSha256(WithFileIdx(localPath, idx)); err != nil {
				localPath = WithFileIdx(localPath, idx)
				fmt.Printf("--> Updating version %d of '%s'\n", idx, localPath)
			} else if hash != entry.SHA256 {
				fmt.Printf("--> Adding version %d to '%s'\n", idx+1, localPath)
				localPath = WithFileIdx(localPath, idx+1)
			} else {
				fmt.Printf("--> Up to date '%s'\n", localPath)
				return nil
			}

		default:
			fmt.Printf("--> Skipping '%s'\n", localPath)
			return nil
		}
	}

	blob, err := client.Download(ctx, entry.Name)
	if err != nil {
		return nil
	}

	localDirName, tempBaseName := path.Split(localPath)
	if err := os.MkdirAll(localDirName, os.ModePerm); err != nil {
		return err
	}

	hasher := sha256.New()

	//	todo: add a progress bar

	tmpFile, err := WriteTempFile(localDirName, tempBaseName, io.TeeReader(blob.ReadCloser, hasher))
	if err != nil {
		return err
	}
	defer tmpFile.Cleanup()

	if hash := hex.EncodeToString(hasher.Sum(nil)); hash != entry.SHA256 {
		return fmt.Errorf("content hash mismatch: expected '%s', have '%s'", entry.SHA256, hash)
	}

	if err := os.Chtimes(tmpFile.Name, blob.Modified, blob.Modified); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name, localPath); err != nil {
		return err
	}

	_ = tmpFile.Release()

	return nil
}
