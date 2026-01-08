package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

type RemoteConfig struct {
	URL string `json:"url"`
	RemoteCredentials
}

type RemoteCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (creds *RemoteCredentials) Equal(other *RemoteCredentials) bool {
	return creds != nil && other != nil &&
		creds.Username == other.Username &&
		creds.Password == other.Password
}

func ParseRemoteUrl(val string) (string, *RemoteCredentials, error) {

	if strings.Contains(val, "://") {

		urlVal, err := url.Parse(val)
		if err != nil {
			return "", nil, err
		}

		var creds *RemoteCredentials
		if urlVal.User != nil && urlVal.User.Username() != "" {
			pass, _ := urlVal.User.Password()
			creds = &RemoteCredentials{
				Username: urlVal.User.Username(),
				Password: pass,
			}
		}

		baseUrl := url.URL{
			Scheme: urlVal.Scheme,
			Host:   urlVal.Host,
			Path:   urlVal.Path,
		}

		return baseUrl.String(), creds, nil
	}

	var pickProto = func(host string) string {

		if hostname, _, err := net.SplitHostPort(host); err == nil {
			host = hostname
		}

		if host == "localhost" || strings.HasSuffix(host, ".local") {
			return "http"
		}

		if ip := net.ParseIP(host); ip != nil {
			if ip.IsPrivate() || ip.IsLoopback() {
				return "http"
			}
		}

		return "https"
	}

	var creds *RemoteCredentials
	if prefix, suffix, ok := strings.Cut(val, "@"); ok {

		credVal, err := ParseRemoteCredentials(prefix)
		if err != nil {
			return "", nil, err
		}

		creds = credVal
		val = suffix
	}

	if prefix, _, ok := strings.Cut(val, "?"); ok {
		val = prefix
	}

	if prefix, _, ok := strings.Cut(val, "#"); ok {
		val = prefix
	}

	host, path, _ := strings.Cut(val, "/")

	return fmt.Sprintf("%s://%s/%s", pickProto(host), host, path), creds, nil
}

func ParseRemoteCredentials(val string) (*RemoteCredentials, error) {

	username, password, ok := strings.Cut(val, ":")
	if !ok || username == "" || password == "" {
		return nil, fmt.Errorf("invalid crednetials string")
	}

	return &RemoteCredentials{
		Username: username,
		Password: password,
	}, nil
}
