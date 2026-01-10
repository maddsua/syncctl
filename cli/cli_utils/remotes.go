package cliutils

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/maddsua/syncctl/cli/config"
	"github.com/maddsua/syncctl/utils"
)

func GetRemote(cfg *config.Config, name string) (config.RemoteConfig, error) {

	if len(cfg.Remotes) == 0 {
		return nil, fmt.Errorf("no remotes configured")
	}

	remote := cfg.Remotes[name]
	if remote.RemoteConfig == nil {
		return nil, errors.New("remote not found")
	}

	return remote.RemoteConfig, nil
}

func ParseRemoteURL(inputURL string) (config.RemoteConfig, error) {

	remoteURL, err := url.Parse(inputURL)
	if err != nil || remoteURL.Scheme == "" || remoteURL.Host == "" {
		return nil, fmt.Errorf("Invalid url argument")
	}

	switch remoteURL.Scheme {

	case "http", "https":

		fmt.Println("Note: Assuming S4 remote url")

		var didYouBringProtection = func() error {

			//	yep he did
			if remoteURL.Scheme == "https" {
				return nil
			}

			//	no need for protection if there's only one person doing it
			if utils.IsLocalHost(remoteURL.Host) {
				return nil
			}

			//	it's mostly idiot-proofing, because if you allow for it - some idiot
			//	will inevitably stick his dingus into the power outlet
			//	and then blame you for all the fireworks
			fmt.Print("\n!!!    Nah, hollon pardner!    !!!\n\n")
			fmt.Println("You're a big boy and shi but don't be silly and use protection")
			return fmt.Errorf("TLS-less connections are not allowed outside of localhost")
		}

		if err := didYouBringProtection(); err != nil {
			return nil, err
		}

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

		return &config.S4RemoteConfig{
			RemoteURL: baseURL.String(),
			Auth:      auth,
		}, nil
	}

	return nil, fmt.Errorf("unsupported url")
}
