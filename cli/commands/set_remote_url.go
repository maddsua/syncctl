package commands

import (
	"fmt"
	"net"
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

		var didYouBringProtection = func() error {

			//	yep he did
			if remoteURL.Scheme == "https" {
				return nil
			}

			//	tbh I wanted to use some corny variable names here as well,
			//	but then it sucks ass to read the code
			hostname := remoteURL.Host
			if val, _, err := net.SplitHostPort(hostname); err == nil {
				hostname = val
			}

			//	no need for protection if there's only one person doing it
			if hostname == "localhost" || net.ParseIP(hostname).IsLoopback() {
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
			return err
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

		cfg.Remote.RemoteConfig = &config.S4RemoteConfig{
			RemoteURL: baseURL.String(),
			Auth:      auth,
		}

		cfg.Changed = true
		return nil
	}

	return fmt.Errorf("Unsupported url")
}
