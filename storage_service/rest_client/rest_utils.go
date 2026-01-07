package rest_client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	s4 "github.com/maddsua/syncctl/storage_service"
)

type NetworkError struct {
	Message       string
	OriginalError error
}

func (err *NetworkError) Error() string {

	if err.OriginalError != nil {
		return fmt.Sprintf("%s: %v", err.Message, err.OriginalError)
	}

	return err.Message
}

func prepareRequest(baseUrl string, auth *url.Userinfo, operationMethod, operationPath string, operationParams url.Values, body io.Reader) (*http.Request, error) {

	requestURL, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	requestURL.Path = path.Join(requestURL.Path, s4.PrefixV1, operationPath)

	if operationParams != nil {
		requestURL.RawQuery = operationParams.Encode()
	}

	req, err := http.NewRequest(operationMethod, requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	if auth != nil {
		password, _ := auth.Password()
		req.SetBasicAuth(auth.Username(), password)
	}

	return req, nil
}

func executeRequest(req *http.Request) (*http.Response, error) {

	response, err := http.DefaultClient.Do(req)
	if err != nil {

		if err, ok := err.(*url.Error); ok {
			return nil, &NetworkError{
				Message:       "api request",
				OriginalError: err,
			}
		}

		return nil, &NetworkError{
			Message:       "http request",
			OriginalError: err,
		}
	}

	return response, nil
}

func executeJSONRequest[R any](req *http.Request) (R, error) {

	response, err := executeRequest(req)
	if err != nil {
		var dummy R
		return dummy, err
	}

	defer response.Body.Close()

	return parseJSONResponse[R](response)
}

func parseJSONResponse[R any](response *http.Response) (R, error) {

	var result s4.APIResponse[R]

	if !strings.Contains(response.Header.Get("Content-Type"), "json") {
		return result.Data, &NetworkError{
			Message:       "api erorr",
			OriginalError: fmt.Errorf("non-json response (status code: %d)", response.StatusCode),
		}
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result.Data, &NetworkError{
			Message:       "decode response json",
			OriginalError: err,
		}
	}

	if result.Error != nil {
		return result.Data, result.Error
	}

	return result.Data, nil
}
