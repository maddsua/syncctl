package utils

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

func ListAllRegularFiles(name string) ([]string, error) {

	if stat, err := os.Stat(name); err != nil {
		if err := os.MkdirAll(name, os.ModePerm); err != nil {
			return nil, err
		}
		return nil, nil
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	var result []string

	entries, err := os.ReadDir(name)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {

		nextName := path.Join(name, entry.Name())

		if entry.Type().IsRegular() {
			result = append(result, nextName)
			continue
		} else if !entry.IsDir() {
			continue
		}

		next, err := ListAllRegularFiles(nextName)
		if len(next) > 0 {
			result = append(result, next...)
		}

		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func WithFileVersion(name string, idx int) string {

	if idx <= 1 {
		return name
	}

	ext := path.Ext(name)
	name = strings.TrimSuffix(name, ext)

	return name + "-" + strconv.Itoa(idx) + ext
}

func NamedFileHighestVersion(name string) (int, error) {

	entries, err := os.ReadDir(path.Dir(name))
	if err != nil {
		return 0, err
	} else if len(entries) < 2 {
		return 1, nil
	}

	indexer := FileVersionIndexer{BaseName: name}

	for _, entry := range entries {
		indexer.Index(entry.Name())
	}

	return indexer.Sum(), nil
}

type FileVersionIndexer struct {
	//	Original (non-versioned) file name
	BaseName string

	prefix, suffix string
	ready          bool
	values         []int
}

func (indexer *FileVersionIndexer) Index(name string) {

	if !indexer.ready {
		indexer.suffix = path.Ext(indexer.BaseName)
		indexer.prefix = strings.TrimSuffix(path.Base(indexer.BaseName), indexer.suffix)
		indexer.ready = true
	}

	name = path.Base(name)

	if !strings.HasPrefix(name, indexer.prefix) || !strings.HasSuffix(name, indexer.suffix) {
		return
	}

	diff := name[len(indexer.prefix):]
	diff = diff[:len(diff)-len(indexer.suffix)]
	if len(diff) < 2 || diff[0] != '-' {
		return
	}

	idx, _ := strconv.Atoi(diff[1:])
	indexer.values = append(indexer.values, idx)
}

func (indexer *FileVersionIndexer) Sum() int {

	if len(indexer.values) == 0 {
		return 1
	}

	slices.Sort(indexer.values)

	return indexer.values[len(indexer.values)-1]
}
