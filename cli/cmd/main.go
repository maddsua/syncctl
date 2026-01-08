package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	//"github.com/maddsua/syncctl/storage_service/rest_client"

	"github.com/maddsua/syncctl/cli"
	"github.com/maddsua/syncctl/storage_service/rest_client"
	metacli "github.com/urfave/cli/v3"
)

/*
	Some cmd examples for sleepy joe:

	pull some shit:
	go run ./cli/cmd pull /docs data/client/docs

*/

//	todo: auth command

//	todo: config commands

func main() {

	//	todo: manage this as well
	client := rest_client.RestClient{
		RemoteURL: "http://localhost:2000/",
	}

	cmd := &metacli.Command{
		Commands: []*metacli.Command{
			{
				Name:  "pull",
				Usage: "Pulls your stupid files from the remote",
				Arguments: []metacli.Argument{
					&metacli.StringArg{
						Name: "remote_dir",
					},
					&metacli.StringArg{
						Name: "local_dir",
					},
				},
				Flags: []metacli.Flag{
					&metacli.BoolFlag{
						Name:  "prune",
						Usage: "Whether or not to nuke all the files that aren't present on the remote",
					},
					&metacli.GenericFlag{
						Name:  "conflict",
						Value: cli.ConflictFlagValue,
						Usage: fmt.Sprintf("How to handle files that already exist locally? [%s]",
							strings.Join(cli.ConflictFlagValue.Options, "|")),
					},
				},
				Action: func(ctx context.Context, cmd *metacli.Command) error {

					//	todo: support setting these from global-ish config
					remoteDir := cmd.StringArg("remote_dir")
					localDir := cmd.StringArg("local_dir")

					if remoteDir == "" && localDir == "" {
						return metacli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if localDir == "" {
						return metacli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := cli.ConflictResolutionPolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := isConflictResolutionConflict(onConflict, prune); err != nil {
						return err
					}

					return pullCmd(ctx, &client, remoteDir, localDir, onConflict, prune)
				},
			},
			{
				Name:  "push",
				Usage: "Pushes your stupid local files to the remote",
				Arguments: []metacli.Argument{
					&metacli.StringArg{
						Name: "local_dir",
					},
					&metacli.StringArg{
						Name: "remote_dir",
					},
				},
				Flags: []metacli.Flag{
					&metacli.BoolFlag{
						Name:  "prune",
						Usage: "Whether or not to nuke all the files that aren't present locally",
					},
					&metacli.GenericFlag{
						Name:  "conflict",
						Value: cli.ConflictFlagValue,
						Usage: fmt.Sprintf("How to handle files that already exist on the remote? [%s]",
							strings.Join(cli.ConflictFlagValue.Options, "|")),
					},
				},
				Action: func(ctx context.Context, cmd *metacli.Command) error {

					//	todo: support setting these from global-ish config
					localDir := cmd.StringArg("local_dir")
					remoteDir := cmd.StringArg("remote_dir")

					if remoteDir == "" && localDir == "" {
						return metacli.Exit("Yo! You forgot to tell the thing where to pull them files from!", 1)
					} else if remoteDir == "" {
						return metacli.Exit("Good job! Now tell it where to put it to!", 1)
					}

					onConflict := cli.ConflictResolutionPolicy(cmd.String("conflict"))
					prune := cmd.Bool("prune")

					if err := isConflictResolutionConflict(onConflict, prune); err != nil {
						return err
					}

					return pushCmd(ctx, &client, localDir, remoteDir, onConflict, prune)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func isConflictResolutionConflict(onConflict cli.ConflictResolutionPolicy, prune bool) error {
	if onConflict == cli.ResolveAsVersions && prune {
		return metacli.Exit("How the fuck do you expect it to keep more than one version while also prunnig everything that's not on the remote?????????????", 1)
	}
	return nil
}
