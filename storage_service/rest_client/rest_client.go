package rest_client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	s4 "github.com/maddsua/syncctl/storage_service"
)

type RestClient struct {
	RemoteURL  string
	Auth       *url.Userinfo
	HttpClient http.Client
}

func (client *RestClient) prepare(ctx context.Context, operationMethod, operationPath string, operationParams url.Values, body io.Reader) (*http.Request, error) {

	requestURL, err := url.Parse(client.RemoteURL)
	if err != nil {
		return nil, err
	}

	requestURL.Path = path.Join(requestURL.Path, s4.UrlPrefixV1, operationPath)

	if operationParams != nil {
		requestURL.RawQuery = operationParams.Encode()
	}

	req, err := http.NewRequest(operationMethod, requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	if client.Auth != nil && client.Auth.Username() != "" {
		password, _ := client.Auth.Password()
		req.SetBasicAuth(client.Auth.Username(), password)
	}

	return req.WithContext(ctx), nil
}

func (client *RestClient) exec(req *http.Request) (*http.Response, error) {

	response, err := client.HttpClient.Do(req)
	if err != nil {

		if err, ok := err.(*url.Error); ok {
			return nil, &NetworkError{
				Message:       "api request",
				OriginalError: errors.Unwrap(err),
			}
		}

		return nil, &NetworkError{
			Message:       "http request",
			OriginalError: err,
		}
	}

	return response, nil
}

func (client *RestClient) Ping(ctx context.Context) error {

	req, err := client.prepare(ctx, http.MethodGet, "/gen_204", nil, nil)
	if err != nil {
		return err
	}

	response, err := client.exec(req)
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

	req, err := client.prepare(ctx, http.MethodPut, "/upload", params, entry.Reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", entry.Size))

	req.Header.Set("Last-Modified", entry.Modified.Format(time.RFC1123))

	if entry.SHA256 != "" {
		req.Header.Set("If-None-Match", "sha256="+entry.SHA256)
	}

	return unwrapJSON[*s4.FileMetadata](client.exec(req))
}

func (client *RestClient) Download(ctx context.Context, name string) (*s4.ReadableFile, error) {

	params := url.Values{}
	params.Set("name", name)

	req, err := client.prepare(ctx, http.MethodGet, "/download", params, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.exec(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {

		if _, err := unwrapJSON[any](response, nil); err != nil {
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

	if val, _ := time.Parse(time.RFC1123, response.Header.Get("Last-Modified")); !val.IsZero() {
		meta.Modified = val
	}

	if val, _ := strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64); val > 0 {
		meta.Size = val
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

	req, err := client.prepare(ctx, http.MethodGet, "/stat", params, nil)
	if err != nil {
		return nil, err
	}

	return unwrapJSON[*s4.FileMetadata](client.exec(req))
}

func (client *RestClient) Move(ctx context.Context, name string, newName string, overwrite bool) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", name)
	params.Set("new_name", newName)

	if overwrite {
		params.Set("overwrite", "true")
	}

	req, err := client.prepare(ctx, http.MethodPost, "/move", params, nil)
	if err != nil {
		return nil, err
	}

	return unwrapJSON[*s4.FileMetadata](client.exec(req))
}

func (client *RestClient) Delete(ctx context.Context, name string) (*s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("name", name)

	req, err := client.prepare(ctx, http.MethodDelete, "/delete", params, nil)
	if err != nil {
		return nil, err
	}

	return unwrapJSON[*s4.FileMetadata](client.exec(req))
}

func (client *RestClient) Find(ctx context.Context, prefix string, recursive bool, offset int, limit int) ([]s4.FileMetadata, error) {

	params := url.Values{}
	params.Set("prefix", prefix)

	if recursive {
		params.Set("recursive", "true")
	}

	req, err := client.prepare(ctx, http.MethodGet, "/find", params, nil)
	if err != nil {
		return nil, err
	}

	return unwrapJSON[[]s4.FileMetadata](client.exec(req))
}
