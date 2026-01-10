package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/maddsua/syncctl"
	cliutils "github.com/maddsua/syncctl/cli/cli_utils"
	"github.com/maddsua/syncctl/cli/commands"
	"github.com/maddsua/syncctl/cli/config"
	"github.com/urfave/cli/v3"
)

func main() {

	var cfg config.Config

	if err := cfg.Load(); err != nil {
		fmt.Println("Load config:", err)
		os.Exit(1)
	}

	var conflictFlagValue = &cliutils.EnumValue{
		Options: []string{
			string(syncctl.ResolveSkip),
			string(syncctl.ResolveOverwrite),
			string(syncctl.ResolveAsCopy),
		},
	}

	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "Pulls your stupid files from the remote",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "remote",
					},
					&cli.StringArg{
						Name: "destination",
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
					&cli.BoolFlag{
						Name:  "dry",
						Usage: "If you want to just watch without touching",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					destinationDir := cmd.StringArg("destination")
					if destinationDir == "" {
						return fmt.Errorf("argument 'destination' not provided")
					}

					remoteArg := cmd.StringArg("remote")
					if remoteArg == "" {
						return fmt.Errorf("argument 'remote' not provided")
					}

					remoteName, remoteDir, ok := strings.Cut(remoteArg, ":")
					if !ok {
						return fmt.Errorf("argument 'remote' must have the following format: 'name:path'")
					}

					remote, err := cliutils.GetRemote(&cfg, remoteName)
					if err != nil {
						return err
					}

					client, err := cliutils.NewS4RestClient(ctx, remote)
					if err != nil {
						return err
					}

					dry := cmd.Bool("dry")

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
						return err
					}

					return commands.Pull(ctx, client, remoteDir, destinationDir, onConflict, prune, dry)
				},
			},
			{
				Name:  "push",
				Usage: "Pushes your stupid local files to the remote",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "source",
					},
					&cli.StringArg{
						Name: "remote",
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
					&cli.BoolFlag{
						Name:  "dry",
						Usage: "If you want to just watch without touching",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					remoteArg := cmd.StringArg("remote")
					if remoteArg == "" {
						return fmt.Errorf("argument 'remote' not provided")
					}

					remoteName, remoteDir, ok := strings.Cut(remoteArg, ":")
					if !ok {
						return fmt.Errorf("argument 'remote' must have the following format: 'name:path'")
					}

					remote, err := cliutils.GetRemote(&cfg, remoteName)
					if err != nil {
						return err
					}

					client, err := cliutils.NewS4RestClient(ctx, remote)
					if err != nil {
						return err
					}

					sourceDir := cmd.StringArg("source")
					if sourceDir == "" {
						return fmt.Errorf("argument 'source' not provided")
					}

					dry := cmd.Bool("dry")

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
						return err
					}

					return commands.Push(ctx, client, sourceDir, remoteDir, onConflict, prune, dry)
				},
			},
			{
				Name:  "remote",
				Usage: "Configure remotes",
				Commands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add or replace a remote",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "name",
							},
							&cli.StringArg{
								Name: "url",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {

							name := cmd.StringArg("name")
							if name == "" {
								return fmt.Errorf("argument 'name' not provided")
							}

							url := cmd.StringArg("url")
							if url == "" {
								return fmt.Errorf("argument 'url' not provided")
							}

							_, existed := cfg.Remotes[name]

							remote, err := cliutils.ParseRemoteURL(url)
							if err != nil {
								return err
							}

							if cfg.Remotes == nil {
								cfg.Remotes = map[string]config.RemoteConfigWrapper{}
							}

							cfg.Remotes[name] = config.RemoteConfigWrapper{RemoteConfig: remote}
							cfg.Changed = true

							if !existed {
								fmt.Println("Remote added")
							} else {
								fmt.Println("Remote updated")
							}

							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "Remove a remote",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "name",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {

							name := cmd.StringArg("name")
							if name == "" {
								return fmt.Errorf("argument 'name' not provided")
							}

							if _, ok := cfg.Remotes[name]; !ok {
								return fmt.Errorf("remote '%s' doesn't exist", name)
							}

							delete(cfg.Remotes, name)
							cfg.Changed = true
							fmt.Println("Remote deleted")

							return nil
						},
					},
					{
						Name:  "status",
						Usage: "Show remote status",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "name",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {

							name := cmd.StringArg("name")
							if name == "" {
								return fmt.Errorf("argument 'name' not provided")
							}

							remote, err := cliutils.GetRemote(&cfg, name)
							if err != nil {
								return err
							}

							fmt.Println("Type:", remote.Type())
							fmt.Println("URL:", remote.URL())

							if remote, ok := remote.(*config.S4RemoteConfig); ok && remote.Auth != nil {
								fmt.Println("User:", remote.Auth.Username)
							} else {
								fmt.Println("[No user set]")
							}

							if _, err := cliutils.NewS4RestClient(ctx, remote); err != nil {
								fmt.Println("Status: Unreachable", err)
							} else {
								fmt.Println("Status: Ready")
							}

							return nil
						},
					},
					{
						Name:  "list",
						Usage: "List remotes",
						Action: func(ctx context.Context, cmd *cli.Command) error {

							fmt.Println("> Config location:", cfg.Location)

							if len(cfg.Remotes) == 0 {
								fmt.Println("[No remotes]")
								return nil
							}

							var names []string
							for key := range cfg.Remotes {
								names = append(names, key)
							}
							slices.Sort(names)

							for _, name := range names {
								remote := cfg.Remotes[name]
								fmt.Printf("%s %s %s\n", name, remote.Type(), remote.URL())
								return nil
							}

							return nil
						},
					},
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
			fmt.Println(err.Error())
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
	if onConflict == syncctl.ResolveAsCopy && prune {
		return fmt.Errorf("Dude did you just set both 'prune' flag and 'copy' conflict resolution strategy together?? Talk about sitting on two chairs with one ass huh?!")
	}
	return nil
}
