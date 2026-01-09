package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/maddsua/syncctl"
	cliutils "github.com/maddsua/syncctl/cli/cli_utils"
	"github.com/maddsua/syncctl/cli/config"
	"github.com/maddsua/syncctl/storage_service/rest_client"
	"github.com/urfave/cli/v3"
)

func main() {

	var cfg config.Config

	if err := cfg.Load(); err != nil {
		fmt.Println("Load config:", err)
		os.Exit(1)
	}

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

					client, err := newS4RestClient(&cfg)
					if err != nil {
						return err
					}

					remoteDir := cmd.StringArg("remote_dir")
					localDir := cmd.StringArg("local_dir")

					if remoteDir == "" && localDir == "" {
						return cli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if localDir == "" {
						return cli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
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

					client, err := newS4RestClient(&cfg)
					if err != nil {
						return err
					}

					localDir := cmd.StringArg("local_dir")
					remoteDir := cmd.StringArg("remote_dir")

					if remoteDir == "" && localDir == "" {
						return cli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if remoteDir == "" {
						return cli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
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

									inputURL := cmd.StringArg("url")
									if inputURL == "" {
										return cli.Exit("Forgot to set the URL itself huh?", 1)
									}

									remoteURL, err := url.Parse(inputURL)
									if err != nil || remoteURL.Scheme == "" || remoteURL.Host == "" {
										return cli.Exit("Invalid url argument", 1)
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

									return cli.Exit("Unsupported url", 1)
								},
							},
						},
					},
				},
			},
			{
				Name:  "status",
				Usage: "Show and check current config",
				Action: func(ctx context.Context, _ *cli.Command) error {

					if !cfg.Valid {
						fmt.Println("[No config found]")
						return nil
					}

					fmt.Println("> Location:", cfg.Location)

					if cfg.Remote.RemoteConfig == nil {
						fmt.Println("[No remote set]")
						return nil
					}

					if _, err := newS4RestClient(&cfg); err != nil {
						return cli.Exit(fmt.Errorf("Unable to configure client: %v", err), 1)
					}

					fmt.Println("> Remote:", cfg.Remote.URL())
					fmt.Println("> Remote type:", cfg.Remote.Type())

					if remote, ok := cfg.Remote.RemoteConfig.(*config.S4RemoteConfig); ok && remote.Auth != nil {
						fmt.Println("> User:", remote.Auth.Username)
					} else {
						fmt.Println("[No user set]")
					}

					return nil
				},
			},
		},
	}

	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := cmd.Run(ctx, os.Args); ctx.Err() == nil {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	exitCh := make(chan os.Signal, 2)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)

	select {

	case err := <-errCh:

		if err != nil {
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

	case <-exitCh:
		fmt.Println("Cancelling...")
		cancel()
		<-errCh
	}
}

func canResolveFileConflicts(onConflict syncctl.ResolvePolicy, prune bool) error {
	if onConflict == syncctl.ResolveAsVersions && prune {
		return cli.Exit("How the fuck do you expect it to keep more than one version while also prunnig everything that's not on the remote?????????????", 1)
	}
	return nil
}

func newS4RestClient(cfg *config.Config) (*rest_client.RestClient, error) {

	if cfg.Remote.RemoteConfig == nil {
		return nil, cli.Exit("Remote not configured. Use 'set remote url' command to set it", 1)
	}

	if remote, ok := cfg.Remote.RemoteConfig.(*config.S4RemoteConfig); ok {

		var check = func(client *rest_client.RestClient) (*rest_client.RestClient, error) {

			if err := client.Ping(context.Background()); err != nil {
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
