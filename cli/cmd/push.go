package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/maddsua/syncctl"
	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/utils"
)

func push_cmd(ctx context.Context, client s4.StorageClient, localDir, remoteDir string, onconflict syncctl.ResolvePolicy, prune, dry bool) error {

	if onconflict == syncctl.ResolveAsCopy {
		prune = false
	}

	fmt.Println("Fetching remote index...")

	remoteIndex := map[string]*s4.FileMetadata{}

	if entries, err := client.Find(ctx, remoteDir, nil, true, 0, 0); err != nil {
		return fmt.Errorf("Unable to fetch remote index: %v", err)
	} else if len(entries) > 0 {
		for _, entry := range entries {
			remoteIndex[entry.Name] = &entry
		}
	}

	fmt.Println("Indexing local files...")

	entries, err := utils.ListRegilarFiles(localDir)
	if err != nil {
		return fmt.Errorf("Unable to list local files: %v", err)
	}

	for _, name := range entries {

		remotePath := path.Join(remoteDir, strings.TrimPrefix(path.Clean(name), path.Clean(localDir)))

		if err := pushEntry(ctx, client, name, remotePath, remoteIndex[remotePath], onconflict, dry); err != nil {
			fmt.Fprintf(os.Stderr, "--X Error pushing '%s':\n", name)
			fmt.Fprintf(os.Stderr, "    %v\n", err)
			return fmt.Errorf("Push aborted")
		}

		delete(remoteIndex, remotePath)
	}

	if prune {
		for key := range remoteIndex {
			if !dry {
				if _, err := client.Delete(ctx, key); err != nil {
					return fmt.Errorf("Unable to prune '%s': %v", key, err)
				}
			}
			fmt.Println("--> Prune", key)
		}
	}

	if !dry {
		fmt.Println("Push complete")
	} else {
		fmt.Println("Dry run (push) complete")
	}

	return nil
}

func pushEntry(ctx context.Context, client s4.StorageClient, name, remotePath string, remoteEntry *s4.FileMetadata, onconflict syncctl.ResolvePolicy, dry bool) error {

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

		if remoteEntry.SHA256 == hash {
			//	debug: log
			//	fmt.Printf("--> Up to date '%s'\n", remotePath)
			return nil
		}

		switch onconflict {

		case syncctl.ResolveSkip:
			fmt.Printf("--> Skip existing '%s' (diff)\n", remotePath)
			return nil

		case syncctl.ResolveAsCopy:

			prefix, basename := path.Split(remotePath)
			baseExt := path.Ext(basename)
			basePrefix := strings.TrimSuffix(basename, baseExt)

			filter := regexp.MustCompile(
				fmt.Sprintf("%s-\\d+%s",
					regexp.QuoteMeta(basePrefix),
					regexp.QuoteMeta(baseExt)))

			entries, err := client.Find(ctx, prefix, filter, false, 0, 0)
			if err != nil {
				return err
			}

			indexer := utils.NewFileVersionIndexer(remotePath)
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
			fmt.Printf("--> Updating '%s' (%s)\n", remotePath, utils.DataSizeString(float64(stat.Size())))
		}

	} else {
		fmt.Printf("--> Uploading '%s' (%s)\n", remotePath, utils.DataSizeString(float64(stat.Size())))
	}

	if !dry {

		//	todo: add a progress bar

		if _, err := client.Put(ctx, &s4.FileUpload{
			FileMetadata: s4.FileMetadata{
				Name:     remotePath,
				Size:     stat.Size(),
				Modified: stat.ModTime(),
			},
			Reader: file,
		}, onconflict == syncctl.ResolveOverwrite); err != nil {
			return err
		}
	}

	return nil
}
