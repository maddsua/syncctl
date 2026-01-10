package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
		Options: []string{string(syncctl.ResolveSkip), string(syncctl.ResolveOverwrite), string(syncctl.ResolveAsCopy)},
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

					client, err := cliutils.NewS4RestClient(ctx, &cfg)
					if err != nil {
						return err
					}

					remoteDir := cmd.StringArg("remote_dir")
					localDir := cmd.StringArg("local_dir")

					if remoteDir == "" && localDir == "" {
						return fmt.Errorf("Yo! You forgot to tell the thing where to pull them files from!")
					} else if localDir == "" {
						return fmt.Errorf("Good job! Now tell it where to put it to!")
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
						return err
					}

					return commands.Pull(ctx, client, remoteDir, localDir, onConflict, prune)
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

					client, err := cliutils.NewS4RestClient(ctx, &cfg)
					if err != nil {
						return err
					}

					localDir := cmd.StringArg("local_dir")
					remoteDir := cmd.StringArg("remote_dir")

					if remoteDir == "" && localDir == "" {
						return fmt.Errorf("Yo! You forgot to tell the thing where to pull them files from!")
					} else if remoteDir == "" {
						return fmt.Errorf("Good job! Now tell it where to put it to!")
					}

					onConflict := syncctl.ResolvePolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := canResolveFileConflicts(onConflict, prune); err != nil {
						return err
					}

					return commands.Push(ctx, client, localDir, remoteDir, onConflict, prune)
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
										return fmt.Errorf("Forgot to set the URL itself huh?")
									}

									return commands.SetRemoteUrl(inputURL, &cfg)
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
					return commands.Status(ctx, &cfg)
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
			fmt.Print("\n")
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
