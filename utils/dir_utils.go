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

	ext := path.Ext(name)
	name = strings.TrimSuffix(name, ext)

	return name + "-" + strconv.Itoa(idx) + ext
}

func HighestFileVersion(name string) (int, error) {

	entries, err := os.ReadDir(path.Dir(name))
	if err != nil {
		return 0, err
	} else if len(entries) < 2 {
		return 0, nil
	}

	ext := path.Ext(name)
	baseName := strings.TrimSuffix(path.Base(name), ext)

	var indexes []int

	for _, entry := range entries {

		name := entry.Name()

		if !strings.HasPrefix(name, baseName) || !strings.HasSuffix(name, ext) {
			continue
		}

		diff := name[len(baseName):]
		diff = diff[:len(diff)-len(ext)]
		if len(diff) < 2 || diff[0] != '-' {
			continue
		}

		idx, _ := strconv.Atoi(diff[1:])
		indexes = append(indexes, idx)
	}

	if len(indexes) < 2 {
		return 0, nil
	}

	slices.Sort(indexes)

	return indexes[len(indexes)-1], nil
}
