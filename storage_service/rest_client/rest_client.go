package rest_client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	s4 "github.com/maddsua/syncctl/storage_service"
)

type RestClient struct {
	RemoteURL string
	Auth      url.Userinfo
}

func (client *RestClient) Ping(ctx context.Context) error {

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodGet, "gen_204", nil, nil)
	if err != nil {
		return err
	}

	response, err := executeRequest(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return &NetworkError{
			Message:       "remote unavailable",
			OriginalError: fmt.Errorf("http status %d", response.StatusCode),
		}
	}

	return nil
}

func (client *RestClient) Put(ctx context.Context, entry *s4.FileUpload, overwrite bool) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", entry.Name)

	if overwrite {
		params.Set("overwrite", "true")
	}

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodPut, "/upload", params, entry.Reader)
	if err != nil {
		return nil, err
	}

	//	todo: fix header name
	req.Header.Set("X-Content-Length", strconv.FormatInt(entry.Size, 10))
	req.Header.Set("Date", entry.Modified.Format(time.RFC1123))

	if entry.SHA256 != "" {
		req.Header.Set("Etag", "sha256="+entry.SHA256)
	}

	return executeJSONRequest[*s4.FileMetadata](req)
}

func (client *RestClient) Download(ctx context.Context, name string) (*s4.ReadableFile, error) {

	params := url.Values{}
	params.Set("name", name)

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodGet, "/download", params, nil)
	if err != nil {
		return nil, err
	}

	response, err := executeRequest(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		if _, err := parseJSONResponse[any](response); err != nil {
			return nil, err
		}
		return nil, &NetworkError{
			Message:       "api error",
			OriginalError: fmt.Errorf("non-json response for a blob erorr"),
		}
	}

	meta := s4.FileMetadata{
		Name: name,
	}

	if val, _ := strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64); val > 0 {
		meta.Size = val
	}

	if val, _ := time.Parse(time.RFC1123, response.Header.Get("Date")); !val.IsZero() {
		meta.Modified = val
	}

	if val, ok := strings.CutPrefix(response.Header.Get("Content-Disposition"), "attachment;"); ok {
		if val, ok = strings.CutPrefix(strings.TrimSpace(val), "filename="); ok {
			if val, _ = url.QueryUnescape(strings.TrimSpace(val)); val != "" {
				meta.Name = val
			}
		}
	}

	if val, ok := strings.CutPrefix(response.Header.Get("Etag"), "sha256="); ok {
		meta.SHA256 = val
	}

	return &s4.ReadableFile{
		FileMetadata: meta,
		ReadCloser:   response.Body,
	}, nil
}

func (client *RestClient) Stat(ctx context.Context, name string) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", name)

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodGet, "/stat", params, nil)
	if err != nil {
		return nil, err
	}

	return executeJSONRequest[*s4.FileMetadata](req)
}

func (client *RestClient) Move(ctx context.Context, name string, newName string, overwrite bool) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", name)
	params.Set("new_name", newName)

	if overwrite {
		params.Set("overwrite", "true")
	}

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodPost, "/move", params, nil)
	if err != nil {
		return nil, err
	}

	return executeJSONRequest[*s4.FileMetadata](req)
}

func (client *RestClient) Delete(ctx context.Context, name string) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", name)

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodDelete, "/delete", params, nil)
	if err != nil {
		return nil, err
	}

	return executeJSONRequest[*s4.FileMetadata](req)
}

func (client *RestClient) List(ctx context.Context, prefix string, recursive bool, offset int, limit int) ([]s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("prefix", prefix)

	if recursive {
		params.Set("recursive", "true")
	}

	req, err := prepareRequest(client.RemoteURL, &client.Auth, http.MethodGet, "/list", params, nil)
	if err != nil {
		return nil, err
	}

	return executeJSONRequest[[]s4.FileMetadata](req)
}
