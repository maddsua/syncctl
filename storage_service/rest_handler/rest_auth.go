package rest_handler

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/maddsua/syncctl/storage_service/config"
)

type AuthThingy struct {
	users sync.Map
}

func (auth *AuthThingy) LoadUsers(users []config.UserConfig) {
	for _, entry := range users {
		auth.users.Store(entry.Username, &UserState{UserConfig: entry})
	}
}

func (auth *AuthThingy) Authorize(req *http.Request) (*UserState, error) {

	creds := extractBasicAuth(req)
	if creds == nil {
		slog.Debug("User auth: Unauthorized")
		return nil, &AuthError{}
	}

	entry, _ := auth.users.Load(creds.Username())
	if state, ok := entry.(*UserState); ok {

		pass, _ := creds.Password()

		if subtle.ConstantTimeCompare([]byte(state.Password), []byte(pass)) != 1 {
			slog.Warn("User auth: Password mismatch",
				slog.String("username", state.Username))
			return nil, &AuthError{IsInvalid: true}
		}

		return state, nil
	}

	slog.Warn("User auth: Username not found",
		slog.String("username", creds.Username()))

	return nil, &AuthError{IsInvalid: true}
}

func extractBasicAuth(req *http.Request) *url.Userinfo {

	schema, value, _ := strings.Cut(strings.TrimSpace(req.Header.Get("Authorization")), " ")
	if !strings.EqualFold(schema, "Basic") {
		return nil
	}

	creds, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil
	}

	name, pass, ok := strings.Cut(string(creds), ":")
	if !ok {
		fmt.Print("ASS")
		return nil
	}

	return url.UserPassword(name, pass)
}

type UserState struct {
	config.UserConfig
}

func (user *UserState) ScopePath(name string) string {
	if user.RootDir == "" {
		return name
	}
	return path.Join(user.RootDir, name)
}

func (user *UserState) UnscopePath(name string) string {
	if user.RootDir == "" {
		return name
	}
	return path.Join("/" + strings.TrimPrefix(path.Clean(name), path.Clean(user.RootDir)))
}

type AuthError struct {
	IsInvalid bool
}

func (err *AuthError) Error() string {
	if err.IsInvalid {
		return "invalid credentials"
	}
	return "unauthorized"
}
