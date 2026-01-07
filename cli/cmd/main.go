package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maddsua/syncctl/cli"
	"github.com/maddsua/syncctl/storage_service/rest_client"
)

func main() {

	client := rest_client.RestClient{
		RemoteURL: "http://localhost:2000/",
	}

	if err := cli.Pull(context.Background(), &client, "/pics", "data/client/pics", cli.ResolveOverwrite, true); err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	/* entries, err := client.List(context.Background(), "", true, 0, 0)
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		fmt.Println(">", entry)
	} */

	/* file, err := client.Download(context.Background(), "/docs/message-to-mr-white.md")
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("FILE", file)

	defer file.Close()

	hasher := sha256.New()

	text, err := io.ReadAll(io.TeeReader(file, hasher))
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("TEXT:", string(text))

	hash := hex.EncodeToString(hasher.Sum(nil))

	fmt.Println("HASH:", hash)

	if hash != file.SHA256 {
		fmt.Println("HASH DIDN'T MATCH")
		os.Exit(2)
	} */

	/* broker := blobstorage.Storage{
		RootDir: "data",
	} */

	/* file, err := broker.Put(&fsserver.FileUpload{
		FileMetadata: fsserver.FileMetadata{
			Name:     "/docs/readme.md",
			Modified: time.Now(),
			Size:     15,
		},
		Reader: bytes.NewReader([]byte("yo sup mr white")),
	}, true)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("FILE", file) */

	/* 	page, err := broker.List("", true, 0, 0)

	   	if err != nil {
	   		fmt.Println("ERR", err)
	   		os.Exit(1)
	   	}

	   	for _, entry := range page {
	   		fmt.Println(">", entry)
	   	} */

	/* 	file, err := broker.Get("/docs/message-to-mr-white.md")
	   	if err != nil {
	   		fmt.Println("ERR", err)
	   		os.Exit(1)
	   	}

	   	fmt.Println("FILE", file)

	   	defer file.Close()
	   	text, err := io.ReadAll(file)
	   	if err != nil {
	   		fmt.Println("ERR", err)
	   		os.Exit(1)
	   	}

	   	fmt.Println("TEXT", string(text)) */

}
