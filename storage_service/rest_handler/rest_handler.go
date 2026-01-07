package rest_handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	s4 "github.com/maddsua/syncctl/storage_service"
)

func NewHandler(storage s4.Storage) s4.SyncHandler {

	//	todo: handle auth

	var wg sync.WaitGroup
	var mux http.ServeMux

	mux.HandleFunc("GET /gen_204", func(wrt http.ResponseWriter, _ *http.Request) {
		wrt.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("PUT /upload", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		meta := s4.FileMetadata{
			Name:     req.URL.Query().Get("name"),
			Modified: time.Now(),
		}

		if val, _ := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64); val > 0 {
			meta.Size = val
		}

		if val, _ := time.Parse(time.RFC1123, req.Header.Get("Date")); !val.IsZero() {
			meta.Modified = val
		}

		if val, ok := strings.CutPrefix(req.Header.Get("Etag"), "sha256="); ok {
			meta.SHA256 = val
		}

		newResponse(storage.Put(req.Context(), &s4.FileUpload{
			FileMetadata: meta,
			Reader:       req.Body,
		}, strings.EqualFold(req.URL.Query().Get("overwrite"), "true"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /download", func(wrt http.ResponseWriter, req *http.Request) {

		//	todo: handle ranges

		wg.Add(1)
		defer wg.Done()

		entry, err := storage.Get(req.Context(), req.URL.Query().Get("name"))
		if err != nil {
			newResponse[any](nil, err).WriteJSON(wrt)
			return
		}

		defer entry.ReadSeekCloser.Close()

		wrt.Header().Set("Content-Type", "application/octet-stream")
		wrt.Header().Set("Content-Length", fmt.Sprint(entry.FileMetadata.Size))
		wrt.Header().Set("Date", entry.FileMetadata.Modified.Format(time.RFC1123))
		wrt.Header().Set("Content-Disposition", "attachment; filename="+url.QueryEscape(entry.Name))
		wrt.Header().Set("Etag", "sha256="+entry.FileMetadata.SHA256)

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
		newResponse(storage.Stat(req.Context(), req.URL.Query().Get("name"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /list", func(wrt http.ResponseWriter, req *http.Request) {

		//	todo: handle pagination

		wg.Add(1)
		defer wg.Done()

		newResponse(storage.List(
			req.Context(),
			req.URL.Query().Get("prefix"),
			strings.EqualFold(req.URL.Query().Get("recursive"), "true"),
			0, 0,
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("POST /move", func(wrt http.ResponseWriter, req *http.Request) {
		newResponse(storage.Move(
			req.Context(),
			req.URL.Query().Get("name"),
			req.URL.Query().Get("new_name"),
			strings.EqualFold(req.URL.Query().Get("overwrite"), "true"),
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("DELETE /delete", func(wrt http.ResponseWriter, req *http.Request) {
		newResponse(storage.Delete(
			req.Context(),
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

func newResponse[T any](val T, err error) *s4.APIResponse[T] {

	var getErrorCode = func(err error) int {

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

	if err != nil {
		return &s4.APIResponse[T]{
			Error: &s4.APIError{
				Message:  err.Error(),
				WithCode: getErrorCode(err),
			},
		}
	}
	return &s4.APIResponse[T]{Data: val}
}
