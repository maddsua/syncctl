package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maddsua/syncctl/fsserver"
)

func NewFsHandler(storage fsserver.Storage) http.Handler {

	//	todo: handle wg
	//	todo: handle auth

	var wg sync.WaitGroup
	var mux http.ServeMux

	mux.HandleFunc("GET /gen_204", func(wrt http.ResponseWriter, _ *http.Request) {
		wrt.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("PUT /upload", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		size, _ := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64)
		modified, _ := time.Parse(time.RFC1123, req.Header.Get("Date"))

		newResponse(storage.Put(&fsserver.FileUpload{
			FileMetadata: fsserver.FileMetadata{
				Name:     req.URL.Query().Get("name"),
				Size:     size,
				Modified: modified,
				SHA256:   req.Header.Get("Etag"),
			},
			Reader: req.Body,
		}, strings.EqualFold(req.URL.Query().Get("overwrite"), "true"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /download", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		entry, err := storage.Get(req.URL.Query().Get("name"))
		if err != nil {
			newResponse[any](nil, err).WriteJSON(wrt)
			return
		}

		defer entry.ReadSeekCloser.Close()

		wrt.Header().Set("Content-Type", "application/octet-stream")
		wrt.Header().Set("Content-Length", fmt.Sprint(entry.FileMetadata.Size))
		wrt.Header().Set("Date", entry.FileMetadata.Modified.Format(time.RFC1123))
		wrt.Header().Set("Etag", entry.FileMetadata.SHA256)

		wrt.WriteHeader(http.StatusOK)

		if _, err := io.Copy(wrt, entry.ReadSeekCloser); err != nil {
			slog.Error("Serve file",
				slog.String("name", entry.Name),
				slog.String("err", err.Error()))
			return
		}

		if flusher, ok := wrt.(http.Flusher); ok {
			flusher.Flush()
		}
	})

	mux.HandleFunc("GET /stat", func(wrt http.ResponseWriter, req *http.Request) {
		newResponse(storage.Stat(req.URL.Query().Get("name"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /list", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		newResponse(storage.List(
			req.URL.Query().Get("prefix"),
			strings.EqualFold(req.URL.Query().Get("recursive"), "true"),
			0, 0,
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("POST /move", func(wrt http.ResponseWriter, req *http.Request) {
		newResponse(storage.Move(
			req.URL.Query().Get("name"),
			req.URL.Query().Get("new_name"),
			strings.EqualFold(req.URL.Query().Get("overwrite"), "true"),
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("DELETE /delete", func(wrt http.ResponseWriter, req *http.Request) {
		newResponse(storage.Delete(
			req.URL.Query().Get("name"),
		)).WriteJSON(wrt)
	})

	return &fsHandler{
		ServeMux:  &mux,
		WaitGroup: &wg,
	}
}

type fsHandler struct {
	*http.ServeMux
	*sync.WaitGroup
}

func newResponse[T any](val T, err error) *fsserver.APIResponse[T] {

	var getErrorCode = func(err error) int {

		switch err.(type) {
		case *fsserver.FileNotFoundError:
			return http.StatusNotFound
		case *fsserver.FileConflictError:
			return http.StatusConflict
		case *fsserver.NameError:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}

	if err != nil {
		return &fsserver.APIResponse[T]{
			Error: &fsserver.APIError{
				Message:  err.Error(),
				WithCode: getErrorCode(err),
			},
		}
	}
	return &fsserver.APIResponse[T]{Data: val}
}
