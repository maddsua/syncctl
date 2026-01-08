package rest_handler

import (
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
	//	todo: add event logging

	var wg sync.WaitGroup
	var mux http.ServeMux

	mux.HandleFunc("GET /gen_204", func(wrt http.ResponseWriter, _ *http.Request) {
		wrt.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("PUT /upload", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		meta := s4.FileMetadata{
			Name: req.URL.Query().Get("name"),
		}

		if val, _ := time.Parse(time.RFC1123, req.Header.Get("Last-Modified")); !val.IsZero() {
			meta.Modified = val
		}

		if val := req.Header.Get("Content-Range"); strings.HasPrefix(val, "bytes") {
			if _, after, ok := strings.Cut(val, "/"); ok {
				if val, _ := strconv.ParseInt(after, 10, 64); val > 0 {
					meta.Size = val
				}
			}
		}

		if val, ok := strings.CutPrefix(req.Header.Get("If-None-Match"), "sha256="); ok {
			meta.SHA256 = val
		}

		genericResponse(storage.Put(req.Context(), &s4.FileUpload{
			FileMetadata: meta,
			Reader:       req.Body,
		}, strings.EqualFold(req.URL.Query().Get("overwrite"), "true"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /download", func(wrt http.ResponseWriter, req *http.Request) {

		wg.Add(1)
		defer wg.Done()

		file, err := storage.Get(req.Context(), req.URL.Query().Get("name"))
		if err != nil {
			errorResponse(err).WriteJSON(wrt)
			return
		}

		defer file.ReadSeekCloser.Close()

		cringe := contentRange{}
		if err := cringe.ParseWith(req.Header.Get("Range"), file.Size); err != nil {
			NewErrorResponseWithCode(err.Error(), http.StatusRequestedRangeNotSatisfiable).WriteJSON(wrt)
			return
		}

		//	static headers that aren't really needed but still are set for informational purposes
		wrt.Header().Set("Content-Type", "application/octet-stream")
		wrt.Header().Set("Accept-Ranges", "bytes")

		//	these are dynamic and slightly repurposed headers
		wrt.Header().Set("Last-Modified", file.FileMetadata.Modified.Format(time.RFC1123))
		wrt.Header().Set("Content-Disposition", "attachment; filename="+url.QueryEscape(file.Name))
		wrt.Header().Set("Etag", "sha256="+file.FileMetadata.SHA256)

		if cringe.Valid {
			wrt.Header().Set("Content-Length", strconv.FormatInt(cringe.Size(), 10))
			wrt.Header().Set("Content-Range", cringe.String())
			wrt.WriteHeader(http.StatusPartialContent)
		} else {
			wrt.Header().Set("Content-Length", strconv.FormatInt(file.FileMetadata.Size, 10))
			wrt.WriteHeader(http.StatusOK)
		}

		if cringe.Valid && cringe.Start > 0 {
			if _, err := file.ReadSeekCloser.Seek(cringe.Start, io.SeekStart); err != nil {
				NewErrorResponseWithCode(err.Error(), http.StatusInternalServerError).WriteJSON(wrt)
			}
		}

		var bodyReader io.Reader = file.ReadSeekCloser
		if cringe.Valid && cringe.End > 0 {
			bodyReader = io.LimitReader(file.ReadSeekCloser, cringe.Size())
		}

		if _, err := io.Copy(wrt, bodyReader); err != nil {
			slog.Error("Serve file",
				slog.String("name", file.Name),
				slog.String("err", err.Error()))
			return
		}

		if flusher, ok := wrt.(http.Flusher); ok {
			flusher.Flush()
		}
	})

	mux.HandleFunc("GET /stat", func(wrt http.ResponseWriter, req *http.Request) {
		genericResponse(storage.Stat(req.Context(), req.URL.Query().Get("name"))).WriteJSON(wrt)
	})

	mux.HandleFunc("GET /list", func(wrt http.ResponseWriter, req *http.Request) {

		limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(req.URL.Query().Get("offset"))

		wg.Add(1)
		defer wg.Done()

		genericResponse(storage.List(
			req.Context(),
			req.URL.Query().Get("prefix"),
			strings.EqualFold(req.URL.Query().Get("recursive"), "true"),
			offset,
			limit,
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("POST /move", func(wrt http.ResponseWriter, req *http.Request) {
		genericResponse(storage.Move(
			req.Context(),
			req.URL.Query().Get("name"),
			req.URL.Query().Get("new_name"),
			strings.EqualFold(req.URL.Query().Get("overwrite"), "true"),
		)).WriteJSON(wrt)
	})

	mux.HandleFunc("DELETE /delete", func(wrt http.ResponseWriter, req *http.Request) {
		genericResponse(storage.Delete(
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
