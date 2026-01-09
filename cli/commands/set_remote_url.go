package commands

import (
	"fmt"
	"net/url"

	"github.com/maddsua/syncctl/cli/config"
)

func SetRemoteUrl(inputURL string, cfg *config.Config) error {

	remoteURL, err := url.Parse(inputURL)
	if err != nil || remoteURL.Scheme == "" || remoteURL.Host == "" {
		return fmt.Errorf("Invalid url argument")
	}

	switch remoteURL.Scheme {

	case "http", "https":

		fmt.Println("Note: Assuming S4 remote url")

		baseURL := url.URL{
			Scheme: remoteURL.Scheme,
			Host:   remoteURL.Host,
			Path:   remoteURL.Path,
		}

		fmt.Println("Setting remote url:", remoteURL)

		var auth *config.S4BasicAuth
		if remoteURL.User != nil && remoteURL.User.Username() != "" {
			pass, _ := remoteURL.User.Password()
			auth = &config.S4BasicAuth{
				Username: remoteURL.User.Username(),
				Password: pass,
			}
			fmt.Println("Setting remote user:", auth.Username)
		}

		cfg.Remote.RemoteConfig = &config.S4RemoteConfig{
			RemoteURL: baseURL.String(),
			Auth:      auth,
		}

		cfg.Changed = true
		return nil
	}

	return fmt.Errorf("Unsupported url")
}
