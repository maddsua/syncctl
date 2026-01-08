package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	//"github.com/maddsua/syncctl/storage_service/rest_client"

	"github.com/maddsua/syncctl"
	cliutils "github.com/maddsua/syncctl/cli/cli_utils"
	"github.com/maddsua/syncctl/cli/config"
	"github.com/maddsua/syncctl/storage_service/rest_client"
	"github.com/urfave/cli/v3"
)

/*
	Some cmd examples for sleepy joe:

	pull some shit:
	go run ./cli/cmd pull /docs data/client/docs

*/

func main() {

	var cfg config.Config

	if err := cfg.Load(); err != nil {
		fmt.Println("Load config:", err)
		os.Exit(1)
	}

	client := instantiateClient(&cfg)

	var conflictFlagValue = &cliutils.EnumValue{
		Options: []string{string(syncctl.ResolveSkip), string(syncctl.ResolveOverwrite), string(syncctl.ResolveAsVersions)},
		Value:   string(syncctl.ResolveSkip),
	}

	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "Pulls your stupid files from the remote",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "remote_dir",
					},
					&cli.StringArg{
						Name: "local_dir",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "prune",
						Usage: "Whether or not to nuke all the files that aren't present on the remote",
					},
					&cli.GenericFlag{
						Name:  "conflict",
						Value: conflictFlagValue,
						Usage: fmt.Sprintf("How to handle files that already exist locally? [%s]",
							strings.Join(conflictFlagValue.Options, "|")),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					if err := isRemoteConfigured(&cfg); err != nil {
						return err
					}

					//	todo: support setting these from global-ish config
					remoteDir := cmd.StringArg("remote_dir")
					localDir := cmd.StringArg("local_dir")

					if remoteDir == "" && localDir == "" {
						return cli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if localDir == "" {
						return cli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := isConflictResolutionConflict(onConflict, prune); err != nil {
						return err
					}

					return pullCmd(ctx, client, remoteDir, localDir, onConflict, prune)
				},
			},
			{
				Name:  "push",
				Usage: "Pushes your stupid local files to the remote",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "local_dir",
					},
					&cli.StringArg{
						Name: "remote_dir",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "prune",
						Usage: "Whether or not to nuke all the files that aren't present locally",
					},
					&cli.GenericFlag{
						Name:  "conflict",
						Value: conflictFlagValue,
						Usage: fmt.Sprintf("How to handle files that already exist on the remote? [%s]",
							strings.Join(conflictFlagValue.Options, "|")),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					if err := isRemoteConfigured(&cfg); err != nil {
						return err
					}

					//	todo: support setting these from global-ish config
					localDir := cmd.StringArg("local_dir")
					remoteDir := cmd.StringArg("remote_dir")

					if remoteDir == "" && localDir == "" {
						return cli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if remoteDir == "" {
						return cli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := isConflictResolutionConflict(onConflict, prune); err != nil {
						return err
					}

					return pushCmd(ctx, client, localDir, remoteDir, onConflict, prune)
				},
			},
			{
				Name:  "set",
				Usage: "Configure settings",
				Commands: []*cli.Command{
					{
						Name:  "remote",
						Usage: "Set remote options",
						Commands: []*cli.Command{
							{
								Name:  "url",
								Usage: "Set remote url",
								Arguments: []cli.Argument{
									&cli.StringArg{
										Name: "url",
									},
								},
								Action: func(ctx context.Context, cmd *cli.Command) error {

									urlArg := cmd.StringArg("url")
									if urlArg == "" {
										return cli.Exit("url must not be empty", 1)
									}

									url, creds, err := config.ParseRemoteUrl(urlArg)
									if err != nil {
										return cli.Exit(fmt.Sprintf("invalid remote url: %v", err), 1)
									}

									fmt.Println("Setting remote url:", url)

									if cfg.Remote.URL != url {

										cfg.Remote.URL = url

										if creds != nil {
											fmt.Println("Setting remote user:", creds.Username)
											cfg.Remote.RemoteCredentials = *creds
										} else {
											cfg.Remote.RemoteCredentials = config.RemoteCredentials{}
											fmt.Println("Note: Don't forget to update your credentials")
										}

										cfg.Changed = true
									}

									return nil
								},
							},
							{
								Name:  "auth",
								Usage: "Set remote credentials",
								Arguments: []cli.Argument{
									&cli.StringArg{
										Name: "credentials",
									},
								},
								Action: func(ctx context.Context, cmd *cli.Command) error {

									credsArg := cmd.StringArg("credentials")
									if credsArg == "" {
										return cli.Exit("credentials string must not be empty", 1)
									}

									newVal, err := config.ParseRemoteCredentials(credsArg)
									if err != nil {
										return cli.Exit(fmt.Sprintf("invalid remote credentials: %v", err), 1)
									}

									fmt.Println("Setting remote credentials")

									if !cfg.Remote.RemoteCredentials.Equal(newVal) {
										cfg.Remote.RemoteCredentials = *newVal
										cfg.Changed = true
									}

									return nil
								},
							},
						},
					},
				},
			},
		},
	}

	//	todo: handle interrupts

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if cfg.Changed {
		if err := cfg.Store(); err != nil {
			fmt.Println("Store config:", err)
			os.Exit(1)
		}
		fmt.Println("Note: Config changed")
	}
}

func isConflictResolutionConflict(onConflict syncctl.ResolvePolicy, prune bool) error {
	if onConflict == syncctl.ResolveAsVersions && prune {
		return cli.Exit("How the fuck do you expect it to keep more than one version while also prunnig everything that's not on the remote?????????????", 1)
	}
	return nil
}

func isRemoteConfigured(cfg *config.Config) error {

	if !cfg.Valid || cfg.Remote.URL == "" {
		return cli.Exit("Remote not configured. Use 'set remote url' command to set it", 1)
	} else if cfg.Remote.RemoteCredentials.Username == "" {
		return cli.Exit("Remote auth not configured. Use 'set remote auth' command to set it", 1)
	}

	return nil
}

func instantiateClient(cfg *config.Config) *rest_client.RestClient {

	if cfg.Remote.RemoteCredentials.Username == "" {
		return &rest_client.RestClient{
			RemoteURL: cfg.Remote.URL,
		}
	}

	return &rest_client.RestClient{
		RemoteURL: cfg.Remote.URL,
		Auth:      url.UserPassword(cfg.Remote.Username, cfg.Remote.Password),
	}
}
