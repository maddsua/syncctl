package rest_handler

import (
	"encoding/json"
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

func writeGeneirc[T any](wrt http.ResponseWriter, val T, err error) error {
	if err != nil {
		return writeError(wrt, err)
	}
	return writeData(wrt, val)
}

func writeData[T any](wrt http.ResponseWriter, val T) error {
	return writeResponse(wrt, s4.APIResponse[T]{Data: val})
}

func writeError(wrt http.ResponseWriter, err error) error {
	switch err := err.(type) {
	case *s4.FileNotFoundError:
		return writeErrorWithCode(wrt, err, http.StatusNotFound)
	case *s4.FileConflictError:
		return writeErrorWithCode(wrt, err, http.StatusConflict)
	case *s4.NameError:
		return writeErrorWithCode(wrt, err, http.StatusBadRequest)
	case *AuthError:

		if !err.IsInvalid {
			wrt.Header().Set("WWW-Authenticate", "Basic")
			return writeErrorWithCode(wrt, err, http.StatusUnauthorized)
		}

		return writeErrorWithCode(wrt, err, http.StatusForbidden)
	default:
		return writeErrorWithCode(wrt, err, http.StatusInternalServerError)
	}
}

func writeErrorWithCode(wrt http.ResponseWriter, err error, code int) error {
	return writeResponse(wrt, s4.APIResponse[any]{
		Error: &s4.APIError{
			Message:  err.Error(),
			WithCode: code,
		},
	})
}

func writeResponse[T any](wrt http.ResponseWriter, resp s4.APIResponse[T]) error {

	wrt.Header().Set("Content-Type", "application/json")

	if resp.Error != nil {
		if resp.Error.WithCode >= http.StatusBadRequest {
			wrt.WriteHeader(resp.Error.WithCode)
		} else {
			wrt.WriteHeader(http.StatusBadRequest)
		}
	}

	enc := json.NewEncoder(wrt)
	enc.SetIndent("", "  ")

	return enc.Encode(resp)
}
