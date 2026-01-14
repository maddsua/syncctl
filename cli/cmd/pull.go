package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/maddsua/syncctl"
	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/utils"
)

func pull_cmd(ctx context.Context, client s4.StorageClient, remoteDir, localDir string, onconflict syncctl.ResolvePolicy, prune, dry bool) error {

	if onconflict == syncctl.ResolveAsCopy {
		prune = false
	}

	pruneMap := map[string]struct{}{}

	if prune {

		fmt.Println("Indexing local files...")

		entries, err := utils.ListRegilarFiles(localDir)
		if err != nil {
			return fmt.Errorf("Unable to list local files: %v", err)
		}

		for _, entry := range entries {
			pruneMap[path.Clean(entry)] = struct{}{}
		}
	}

	fmt.Println("Fetching remote index...")

	remoteFiles, err := client.Find(ctx, remoteDir, nil, true, 0, 0)
	if err != nil {
		return fmt.Errorf("Unable to fetch remote index: %v", err)
	} else if len(remoteFiles) == 0 {
		fmt.Println("No files on the remote. Exiting.")
		return nil
	}

	for _, entry := range remoteFiles {

		localPath := path.Join(localDir, strings.TrimPrefix(path.Clean(entry.Name), path.Clean(remoteDir)))

		if err := pullEntry(ctx, client, localPath, onconflict, &entry, dry); err != nil {
			fmt.Printf("--X Error pulling '%s':\n", entry.Name)
			fmt.Printf("    %v\n", err)
			return fmt.Errorf("Pull aborted")
		}

		delete(pruneMap, localPath)
	}

	if prune {
		for name := range pruneMap {
			if !dry {
				if err := os.Remove(name); err != nil {
					return fmt.Errorf("Unable to prune '%s': %v", name, err)
				}
			}
			fmt.Println("--> Prune", name)
		}
	}

	if !dry {
		fmt.Println("Pull complete")
	} else {
		fmt.Println("Dry run (pull) complete")
	}

	return nil
}

func pullEntry(ctx context.Context, client s4.StorageClient, localPath string, onconflict syncctl.ResolvePolicy, entry *s4.FileMetadata, dry bool) error {

	if stat, _ := os.Stat(localPath); stat != nil {

		hash, err := utils.NamedFileHashSha256(localPath)
		if err != nil {
			return err
		}

		if hash == entry.SHA256 {
			//	debug: log
			//	fmt.Printf("--> Up to date '%s'\n", localPath)
			return nil
		}

		switch onconflict {

		case syncctl.ResolveSkip:
			fmt.Printf("--> Skip existing '%s' (diff)\n", localPath)
			return nil

		case syncctl.ResolveAsCopy:

			entries, err := os.ReadDir(path.Dir(localPath))
			if err != nil {
				return err
			}

			indexer := utils.NewFileVersionIndexer(localPath)
			for _, entry := range entries {
				indexer.Index(entry.Name())
			}

			version := indexer.Sum()
			latest := utils.WithFileVersion(localPath, version)

			if hash, err := utils.NamedFileHashSha256(latest); err != nil {
				return fmt.Errorf("hash '%s': %v", latest, err)
			} else if hash != entry.SHA256 {
				fmt.Printf("--> Adding version %d to '%s'\n", version+1, localPath)
				localPath = utils.WithFileVersion(localPath, version+1)
			} else {
				fmt.Printf("--> Up to date '%s', version %d\n", localPath, version)
				return nil
			}

		default:
			fmt.Printf("--> Updating '%s' (%s)\n", localPath, utils.DataSizeString(float64(entry.Size)))
		}

	} else {
		fmt.Printf("--> Downloading '%s' (%s)\n", localPath, utils.DataSizeString(float64(entry.Size)))
	}

	if !dry {

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

		tmpFile, err := utils.WriteTempFile(localDirName, tempBaseName, io.TeeReader(blob.ReadCloser, hasher))
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
	}

	return nil
}
