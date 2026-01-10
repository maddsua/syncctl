package rest_client

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func unwrapJSON[R any](response *http.Response, err error) (R, error) {

	var result s4.APIResponse[R]
	if err != nil {
		return result.Data, err
	}
	defer response.Body.Close()

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
