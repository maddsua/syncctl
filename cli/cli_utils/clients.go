package cliutils

import (
	"context"
	"fmt"
	"net/url"

	"github.com/maddsua/syncctl/cli/config"
	"github.com/maddsua/syncctl/storage_service/rest_client"
)

func NewS4RestClient(ctx context.Context, cfg *config.Config) (*rest_client.RestClient, error) {

	if cfg.Remote.RemoteConfig == nil {
		return nil, fmt.Errorf("Remote not configured. Use 'set remote url' command to set it")
	}

	if remote, ok := cfg.Remote.RemoteConfig.(*config.S4RemoteConfig); ok {

		var check = func(client *rest_client.RestClient) (*rest_client.RestClient, error) {

			if err := client.Ping(ctx); err != nil {
				return client, fmt.Errorf("ping: %v", err)
			}

			return client, nil
		}

		if remote.Auth != nil {
			return check(&rest_client.RestClient{
				RemoteURL: remote.RemoteURL,
				Auth:      url.UserPassword(remote.Auth.Username, remote.Auth.Password),
			})
		}

		return check(&rest_client.RestClient{
			RemoteURL: remote.RemoteURL,
		})
	}

	return nil, fmt.Errorf("unsupported remote type")
}
