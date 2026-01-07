package storage_service

import (
	"encoding/json"
	"net/http"
)

const PrefixV1 = "/s4/v1/"

type APIResponse[T any] struct {
	Data  T         `json:"data"`
	Error *APIError `json:"error"`
}

func (response *APIResponse[T]) WriteJSON(wrt http.ResponseWriter) {

	wrt.Header().Set("Content-Type", "application/json")

	if response.Error != nil {
		if response.Error.WithCode >= http.StatusBadRequest {
			wrt.WriteHeader(response.Error.WithCode)
		} else {
			wrt.WriteHeader(http.StatusBadRequest)
		}
	}

	enc := json.NewEncoder(wrt)
	enc.SetIndent("", "  ")

	_ = enc.Encode(response)
}

type APIError struct {
	Message  string `json:"message"`
	WithCode int    `json:"-"`
}

func (err *APIError) Error() string {
	return err.Message
}
