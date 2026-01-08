package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

//	todo: fix error handling

func Push(ctx context.Context, client s4.StorageClient, localDir, remoteDir string, onconflict ConflictResolutionPolicy, prune bool) error {

	if onconflict == ResolveAsVersions {
		prune = false
	}

	fmt.Println("Fetching remote index...")

	remoteIndex := map[string]*s4.FileMetadata{}

	if entries, err := client.List(ctx, remoteDir, true, 0, 0); err != nil {
		return err
	} else if len(entries) > 0 {
		for _, entry := range entries {
			remoteIndex[entry.Name] = &entry
		}
	}

	fmt.Println("Indexing local files...")

	entries, err := ListAllRegularFiles(localDir)
	if err != nil {
		return err
	}

	for _, name := range entries {

		remotePath := path.Join(remoteDir, strings.TrimPrefix(path.Clean(name), path.Clean(localDir)))

		if err := pushEntry(ctx, client, name, remotePath, remoteIndex[remotePath], onconflict); err != nil {
			fmt.Fprintf(os.Stderr, "--X Error pushing '%s':\n", name)
			fmt.Fprintf(os.Stderr, "    %v\n", err)
			return err
		}

		delete(remoteIndex, remotePath)
	}

	if prune {
		for key := range remoteIndex {
			if _, err := client.Delete(ctx, key); err != nil {
				return err
			}
			fmt.Println("--> Prune", key)
		}
	}

	fmt.Println("Push complete")

	return nil
}

func pushEntry(ctx context.Context, client s4.StorageClient, name, remotePath string, remoteEntry *s4.FileMetadata, onconflict ConflictResolutionPolicy) error {

	stat, err := os.Stat(name)
	if err != nil {
		return err
	}

	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	hash, err := FileHashSha256(file)
	if err != nil {
		return err
	}

	if remoteEntry != nil {

		switch onconflict {

		case ResolveOverwrite:

			if remoteEntry.SHA256 == hash && remoteEntry.Modified.Equal(stat.ModTime()) {
				fmt.Printf("--> Up to date '%s'\n", remotePath)
				return nil
			}

			fmt.Printf("--> Updating '%s' (%s)\n", remotePath, DataSizeString(float64(stat.Size())))

		case ResolveAsVersions:

			//	todo: check naming and shit
			return fmt.Errorf("'ResolveAsVersions' not implemented yet")

		default:
			fmt.Printf("--> Skipping '%s'\n", remotePath)
			return nil
		}
	} else {
		fmt.Printf("--> Uploading '%s' (%s)\n", remotePath, DataSizeString(float64(stat.Size())))
	}

	//	todo: add a progress bar

	if _, err := client.Put(ctx, &s4.FileUpload{
		FileMetadata: s4.FileMetadata{
			Name:     remotePath,
			Size:     stat.Size(),
			Modified: stat.ModTime(),
		},
		Reader: file,
	}, true); err != nil {
		return err
	}

	return nil
}
