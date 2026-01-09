package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/maddsua/syncctl"
	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/utils"
	metacli "github.com/urfave/cli/v3"
)

func pushCmd(ctx context.Context, client s4.StorageClient, localDir, remoteDir string, onconflict syncctl.ResolvePolicy, prune bool) error {

	if onconflict == syncctl.ResolveAsCopy {
		prune = false
	}

	fmt.Println("Fetching remote index...")

	remoteIndex := map[string]*s4.FileMetadata{}

	if entries, err := client.List(ctx, remoteDir, true, 0, 0); err != nil {
		return metacli.Exit(fmt.Sprintf("Unable to fetch remote index: %v", err), 1)
	} else if len(entries) > 0 {
		for _, entry := range entries {
			remoteIndex[entry.Name] = &entry
		}
	}

	fmt.Println("Indexing local files...")

	entries, err := utils.ListAllRegularFiles(localDir)
	if err != nil {
		return metacli.Exit(fmt.Sprintf("Unable to list local files: %v", err), 1)
	}

	for _, name := range entries {

		remotePath := path.Join(remoteDir, strings.TrimPrefix(path.Clean(name), path.Clean(localDir)))

		if err := pushEntry(ctx, client, name, remotePath, remoteIndex[remotePath], onconflict); err != nil {
			fmt.Fprintf(os.Stderr, "--X Error pushing '%s':\n", name)
			fmt.Fprintf(os.Stderr, "    %v\n", err)
			return metacli.Exit("Push aborted", 1)
		}

		delete(remoteIndex, remotePath)
	}

	if prune {
		for key := range remoteIndex {
			if _, err := client.Delete(ctx, key); err != nil {
				return metacli.Exit(fmt.Sprintf("Unable to prune '%s': %v", key, err), 1)
			}
			fmt.Println("--> Prune", key)
		}
	}

	fmt.Println("Push complete")

	return nil
}

func pushEntry(ctx context.Context, client s4.StorageClient, name, remotePath string, remoteEntry *s4.FileMetadata, onconflict syncctl.ResolvePolicy) error {

	stat, err := os.Stat(name)
	if err != nil {
		return err
	}

	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	hash, err := utils.FileHashSha256(file)
	if err != nil {
		return err
	}

	if remoteEntry != nil {

		switch onconflict {

		case syncctl.ResolveOverwrite:

			if remoteEntry.SHA256 == hash && remoteEntry.Modified.Equal(stat.ModTime()) {
				fmt.Printf("--> Up to date '%s'\n", remotePath)
				return nil
			}

			fmt.Printf("--> Updating '%s' (%s)\n", remotePath, utils.DataSizeString(float64(stat.Size())))

		case syncctl.ResolveAsCopy:

			prefix := strings.TrimSuffix(remotePath, path.Ext(remotePath))

			entries, err := client.List(ctx, prefix, false, 0, 0)
			if err != nil {
				return err
			}

			indexer := utils.FileVersionIndexer{BaseName: remotePath}
			for _, entry := range entries {
				indexer.Index(entry.Name)
			}

			version := indexer.Sum()
			latest := utils.WithFileVersion(remotePath, version)

			if stat, err := client.Stat(ctx, latest); err != nil {
				return fmt.Errorf("remote stat '%s': %v", latest, err)
			} else if stat.SHA256 != hash {
				fmt.Printf("--> Adding version %d to '%s'\n", version+1, remotePath)
				remotePath = utils.WithFileVersion(remotePath, version+1)
			} else {
				fmt.Printf("--> Up to date '%s', version %d\n", remotePath, version)
				return nil
			}

		default:
			return nil
		}
	} else {
		fmt.Printf("--> Uploading '%s' (%s)\n", remotePath, utils.DataSizeString(float64(stat.Size())))
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
