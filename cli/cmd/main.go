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

//	todo: push command

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
						Usage: fmt.Sprintf("What do when a file with the same name already exists [%s]",
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

					return cli.Pull(
						ctx,
						&client,
						remoteDir,
						localDir,
						cli.ConflictResolutionPolicy(cmd.String("conflict")),
						cmd.Bool("prune"),
					)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
