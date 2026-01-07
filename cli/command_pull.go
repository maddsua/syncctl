package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

func Pull(ctx context.Context, client s4.StorageClient, remoteDir, localDir string, onconflict FileConflicResolution, prune bool) error {

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

func pullEntry(ctx context.Context, client s4.StorageClient, localPath string, onconflict FileConflicResolution, entry *s4.FileMetadata) error {

	if info, err := FileExists(localPath); err != nil {
		return err
	} else if info != nil {

		switch onconflict {

		case ResolveOverwrite:

			if info.SHA256 == entry.SHA256 {

				fmt.Println("No changes for", localPath)

				if !info.Modified.Equal(entry.Modified) {
					fmt.Println("Update mtime for", localPath)
					if err := os.Chtimes(localPath, entry.Modified, entry.Modified); err != nil {
						return err
					}
				}

				return nil
			}

			fmt.Println("Updating", localPath)

		case ResolveStoreBoth:
			//	todo: update name
			return fmt.Errorf("renaming is not implemented yet")

		default:
			fmt.Println("Skipping existing", localPath)
			return nil
		}

	} else {
		fmt.Println("Copying", localPath)
	}

	blob, err := client.Download(ctx, entry.Name)
	if err != nil {
		return nil
	}

	//	todo: add progress

	if err := WriteLocalFile(localPath, blob.ReadCloser, blob.Modified); err != nil {
		return err
	}

	return nil
}
