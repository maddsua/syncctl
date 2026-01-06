package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/maddsua/syncctl/fsserver"
	"github.com/maddsua/syncctl/fsserver/blobstorage"
)

func main() {

	broker := blobstorage.Storage{
		RootDir: "data",
	}

	file, err := broker.Put(&fsserver.FileUpload{
		FileMetadata: fsserver.FileMetadata{
			Name:     "/docs/message-to-mr-white.md",
			Modified: time.Now(),
			Size:     15,
		},
		Reader: bytes.NewReader([]byte("yo sup mr white")),
	}, true)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("FILE", file)

	/* page, err := broker.List(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	for _, entry := range page.Entries {
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
