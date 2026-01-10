package cliutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"github.com/maddsua/syncctl/cli/config"
	"github.com/maddsua/syncctl/storage_service/rest_client"
	"github.com/maddsua/syncctl/utils"
)

func NewS4RestClient(ctx context.Context, cfg config.RemoteConfig) (*rest_client.RestClient, error) {

	if remote, ok := cfg.(*config.S4RemoteConfig); ok {

		client := rest_client.RestClient{
			RemoteURL: remote.RemoteURL,
		}

		if url, err := url.Parse(client.RemoteURL); err != nil {
			return nil, fmt.Errorf("invalid remote url")
		} else if url.Scheme == "https" && (utils.IsLocalHost(url.Host) || utils.IsLocalNetwork(url.Host)) {
			//	disable certificate verification on localhost and local networks.
			//	doesn't look awfully secure but it'll do for now
			//	todo: configure behavior
			client.HttpClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}

		if remote.Auth != nil {
			client.Auth = url.UserPassword(remote.Auth.Username, remote.Auth.Password)
		}

		if err := client.Ping(ctx); err != nil {
			return nil, fmt.Errorf("ping: %v", err)
		}

		return &client, nil
	}

	return nil, fmt.Errorf("unsupported remote type")
}
