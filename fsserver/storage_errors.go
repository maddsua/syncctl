package fsserver

type StorageError struct {
	Message string `json:"message"`
}

func (err *StorageError) Error() string {
	return err.Message
}

var ErrNoFile = &StorageError{
	Message: "file not found",
}

var ErrFileConflict = &StorageError{
	Message: "file conflict",
}

var ErrInvalidFileName = &StorageError{
	Message: "invalid file name",
}
