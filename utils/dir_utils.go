package utils

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

func ListRegilarFiles(name string) ([]string, error) {

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

		if entry.Type().IsRegular() && NameListable(entry.Name()) {
			result = append(result, nextName)
		} else if entry.IsDir() {

			next, err := ListRegilarFiles(nextName)
			if len(next) > 0 {
				result = append(result, next...)
			}

			if err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

func NameListable(name string) bool {

	switch runtime.GOOS {
	case "android", "linux":
		return !strings.HasPrefix(path.Base(name), ".trashed-")
	}

	return true
}

func WithFileVersion(name string, idx int) string {

	if idx <= 1 {
		return name
	}

	ext := path.Ext(name)
	name = strings.TrimSuffix(name, ext)

	return name + "-" + strconv.Itoa(idx) + ext
}

type FileVersionIndexer interface {
	Index(name string)
	Sum() int
}

func NewFileVersionIndexer(baseName string) FileVersionIndexer {
	suffix := path.Ext(baseName)
	return &fileVersionIndexerImpl{
		suffix: suffix,
		prefix: strings.TrimSuffix(path.Base(baseName), suffix),
	}
}

type fileVersionIndexerImpl struct {
	prefix, suffix string
	values         []int
}

func (indexer *fileVersionIndexerImpl) Index(name string) {

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

func (indexer *fileVersionIndexerImpl) Sum() int {

	if len(indexer.values) == 0 {
		return 1
	}

	slices.Sort(indexer.values)

	return indexer.values[len(indexer.values)-1]
}
