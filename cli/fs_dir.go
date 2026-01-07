package cli

import (
	"fmt"
	"os"
	"path"
)

func ListAllRegular(name string) ([]string, error) {

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

		next, err := ListAllRegular(nextName)
		if len(next) > 0 {
			result = append(result, next...)
		}

		if err != nil {
			return result, err
		}
	}

	return result, nil
}
