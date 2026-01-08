package storage_service

import (
	"net/http"
)

const UrlPrefixV1 = "/s4/v1/"

type SyncHandler interface {
	http.Handler
	Wait()
}

type APIResponse[T any] struct {
	Data  T         `json:"data"`
	Error *APIError `json:"error"`
}

type APIError struct {
	Message  string `json:"message"`
	WithCode int    `json:"-"`
}

func (err *APIError) Error() string {
	return err.Message
}
