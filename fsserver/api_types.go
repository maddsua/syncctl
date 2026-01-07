package fsserver

import (
	"encoding/json"
	"net/http"
)

type APIResponse[T any] struct {
	Data  T         `json:"data"`
	Error *APIError `json:"error"`
}

func (response *APIResponse[T]) WriteJSON(wrt http.ResponseWriter) {

	wrt.Header().Set("Content-Type", "application/json")

	if response.Error != nil {
		wrt.WriteHeader(http.StatusBadRequest)
	}

	enc := json.NewEncoder(wrt)
	enc.SetIndent("", "  ")

	_ = enc.Encode(response)
}

type APIError struct {
	Message string `json:"message"`
}
