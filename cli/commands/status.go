package commands

import (
	"fmt"

	cliutils "github.com/maddsua/syncctl/cli/cli_utils"
	"github.com/maddsua/syncctl/cli/config"
)

func Status(cfg *config.Config) error {

	if !cfg.Valid {
		fmt.Println("[No config found]")
		return nil
	}

	fmt.Println("> Location:", cfg.Location)

	if cfg.Remote.RemoteConfig == nil {
		fmt.Println("[No remote set]")
		return nil
	}

	if _, err := cliutils.NewS4RestClient(cfg); err != nil {
		return fmt.Errorf("Unable to configure client: %v", err)
	}

	fmt.Println("> Remote:", cfg.Remote.URL())
	fmt.Println("> Remote type:", cfg.Remote.Type())

	if remote, ok := cfg.Remote.RemoteConfig.(*config.S4RemoteConfig); ok && remote.Auth != nil {
		fmt.Println("> User:", remote.Auth.Username)
	} else {
		fmt.Println("[No user set]")
	}

	return nil
}
