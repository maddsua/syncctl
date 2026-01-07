package utils

import "os"

type FileJanitor struct {
	Name string

	//	A flag to tell this cleanup thingy to fuck off.
	//	Not using atomic values here since it's not intended for concurrent execution,
	// 	but rather to avoid variable fuckery inside function body
	released bool
}

func (janitor *FileJanitor) Release() string {
	janitor.released = true
	return janitor.Name
}

func (janitor *FileJanitor) Cleanup() error {
	if !janitor.released {
		return os.Remove(janitor.Name)
	}
	return nil
}
