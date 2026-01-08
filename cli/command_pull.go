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

//	todo: wrap all of this into a proper command
//	todo: color the output. maybe with: https://github.com/charmbracelet/lipgloss

func Pull(ctx context.Context, client s4.StorageClient, remoteDir, localDir string, onconflict ConflictResolutionPolicy, prune bool) error {

	if onconflict == ResolveAsVersions && prune {
		return fmt.Errorf("can't use both file versioning and prunning at the same time! wtf?!")
	}

	var pruneMap map[string]struct{}

	if prune {

		fmt.Println("Indexing local files at", localDir)

		entries, err := ListAllRegular(localDir)
		if err != nil {
			return err
		}

		pruneMap = map[string]struct{}{}

		for _, entry := range entries {
			pruneMap[path.Clean(entry)] = struct{}{}
		}
	}

	fmt.Println("Fetching remote index for", remoteDir)

	remoteFiles, err := client.List(ctx, remoteDir, true, 0, 0)
	if err != nil {
		return err
	} else if len(remoteFiles) == 0 {
		fmt.Println("No files on the remote")
		return nil
	}

	for _, entry := range remoteFiles {

		localPath := path.Join(localDir, strings.TrimPrefix(path.Clean(entry.Name), path.Clean(remoteDir)))

		if err := pullEntry(ctx, client, localPath, onconflict, &entry); err != nil {
			return err
		}

		delete(pruneMap, localPath)
	}

	if prune && pruneMap != nil {

		for name := range pruneMap {
			fmt.Println("Prune", name)
			if err := os.Remove(name); err != nil {
				return err
			}
		}
	}

	return nil
}

func pullEntry(ctx context.Context, client s4.StorageClient, localPath string, onconflict ConflictResolutionPolicy, entry *s4.FileMetadata) error {

	if stat, err := FileContentStat(localPath); err != nil {
		return err
	} else if stat == nil {
		fmt.Println("Copying", localPath)
	} else {

		switch onconflict {

		case ResolveOverwrite:

			if stat.SHA256 == entry.SHA256 {

				fmt.Println("No changes for", localPath)

				if !stat.Modified.Equal(entry.Modified) {
					fmt.Println("Update mtime for", localPath)
					if err := os.Chtimes(localPath, entry.Modified, entry.Modified); err != nil {
						return err
					}
				}

				return nil
			}

			fmt.Println("Updating", localPath)

		case ResolveAsVersions:

			if stat.SHA256 == entry.SHA256 {
				fmt.Println("No changes for", localPath)
				return nil
			}

			idx, err := HighestFileIndex(localPath)
			if err != nil {
				return err
			}

			if hash, err := FileSha256HashString(WithFileIdx(localPath, idx)); err != nil {
				localPath = WithFileIdx(localPath, idx)
				fmt.Println("Updating", localPath, "version", idx)
			} else if hash != entry.SHA256 {
				fmt.Println("Adding version", localPath, idx+1)
				localPath = WithFileIdx(localPath, idx+1)
			} else {
				fmt.Println("No new versions", localPath)
				return nil
			}

		default:
			fmt.Println("Skipping existing", localPath)
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

	//	todo: add progress

	tmpFile, err := WriteTempFile(localDirName, tempBaseName, io.TeeReader(blob.ReadCloser, hasher))
	if err != nil {
		return err
	}
	defer tmpFile.Cleanup()

	if hash := hex.EncodeToString(hasher.Sum(nil)); hash != entry.SHA256 {
		return fmt.Errorf("Hash mismatch: '%s' instead of '%s'", hash, entry.SHA256)
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
