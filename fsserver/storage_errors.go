package fsserver

import "fmt"

type FileNotFoundError struct {
	Path string
}

func (err *FileNotFoundError) Error() string {
	return fmt.Sprintf("file '%s' not found", err.Path)
}

type FileConflictError struct {
	Path string
}

func (err *FileConflictError) Error() string {
	return fmt.Sprintf("file '%s' already exists", err.Path)
}

type NameError struct {
	Name string
}

func (err *NameError) Error() string {
	return fmt.Sprintf("file name '%s' invalid", err.Name)
}
