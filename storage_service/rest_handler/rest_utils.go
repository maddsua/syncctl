package rest_handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

type ByteRange struct {
	Start, End, TotalSize int64
	Valid                 bool
}

func (byterange *ByteRange) Size() int64 {
	return byterange.End - byterange.Start
}

func (byterange *ByteRange) String() string {
	return fmt.Sprintf("bytes %d-%d/%d", byterange.Start, byterange.End, byterange.TotalSize)
}

func (byterange *ByteRange) ParseWith(val string, totalSize int64) error {

	if val == "" {
		*byterange = ByteRange{}
		return nil
	}

	const prefix = "bytes="

	if !strings.HasPrefix(val, prefix) {
		return fmt.Errorf("invalid range type")
	} else if strings.Contains(val, ",") {
		return fmt.Errorf("only one range at the time is supported")
	}

	val = strings.TrimSpace(val[len(prefix):])

	before, after, ok := strings.Cut(val, "-")
	if !ok {
		return fmt.Errorf("invalid range value")
	}

	byterange.TotalSize = totalSize

	if before != "" {
		byterange.Start, _ = strconv.ParseInt(before, 10, 64)
		if byterange.Start < 0 {
			return fmt.Errorf("invalid range start")
		} else if byterange.Start >= byterange.TotalSize {
			return fmt.Errorf("range start exceeds file size")
		}
	}

	if after != "" {
		byterange.End, _ = strconv.ParseInt(after, 10, 64)
		if byterange.End < 0 {
			return fmt.Errorf("invalid range end")
		} else if byterange.TotalSize > 0 && byterange.End > byterange.TotalSize {
			return fmt.Errorf("range end exceeds file size")
		} else if byterange.Start >= byterange.End {
			return fmt.Errorf("range start and end are overlapping")
		}
	}

	if byterange.End == 0 {
		byterange.End = byterange.TotalSize
	}

	byterange.Valid = true

	return nil
}

func GetErrorCode(err error) int {

	switch err.(type) {
	case *s4.FileNotFoundError:
		return http.StatusNotFound
	case *s4.FileConflictError:
		return http.StatusConflict
	case *s4.NameError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func NewGenericResponse[T any](val T, err error) *s4.APIResponse[T] {

	if err != nil {
		return &s4.APIResponse[T]{
			Error: &s4.APIError{
				Message:  err.Error(),
				WithCode: GetErrorCode(err),
			},
		}
	}
	return &s4.APIResponse[T]{Data: val}
}

func NewErrorResponse(err error) *s4.APIResponse[any] {
	return &s4.APIResponse[any]{
		Error: &s4.APIError{
			Message:  err.Error(),
			WithCode: GetErrorCode(err),
		},
	}
}

func NewErrorResponseWithCode(message string, code int) *s4.APIResponse[any] {
	return &s4.APIResponse[any]{
		Error: &s4.APIError{
			Message:  message,
			WithCode: code,
		},
	}
}
