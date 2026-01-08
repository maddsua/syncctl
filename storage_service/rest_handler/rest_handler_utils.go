package rest_handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

type contentRange struct {
	Start, End, TotalSize int64
	Valid                 bool
}

func (cr *contentRange) Size() int64 {
	return cr.End - cr.Start
}

func (cr *contentRange) String() string {
	return fmt.Sprintf("bytes %d-%d/%d", cr.Start, cr.End, cr.TotalSize)
}

func (cr *contentRange) ParseWith(val string, totalSize int64) error {

	if val == "" {
		*cr = contentRange{}
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

	cr.TotalSize = totalSize

	if before != "" {
		cr.Start, _ = strconv.ParseInt(before, 10, 64)
		if cr.Start < 0 {
			return fmt.Errorf("invalid range start")
		} else if cr.Start >= cr.TotalSize {
			return fmt.Errorf("range start exceeds file size")
		}
	}

	if after != "" {
		cr.End, _ = strconv.ParseInt(after, 10, 64)
		if cr.End < 0 {
			return fmt.Errorf("invalid range end")
		} else if cr.TotalSize > 0 && cr.End > cr.TotalSize {
			return fmt.Errorf("range end exceeds file size")
		} else if cr.Start >= cr.End {
			return fmt.Errorf("range start and end are overlapping")
		}
	}

	if cr.End == 0 {
		cr.End = cr.TotalSize
	}

	cr.Valid = true

	return nil
}

func errorCode(err error) int {

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

func genericResponse[T any](val T, err error) *s4.APIResponse[T] {

	if err != nil {
		return &s4.APIResponse[T]{
			Error: &s4.APIError{
				Message:  err.Error(),
				WithCode: errorCode(err),
			},
		}
	}
	return &s4.APIResponse[T]{Data: val}
}

func errorResponse(err error) *s4.APIResponse[any] {
	return &s4.APIResponse[any]{
		Error: &s4.APIError{
			Message:  err.Error(),
			WithCode: errorCode(err),
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
