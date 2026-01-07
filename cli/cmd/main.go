package main

import (
	"fmt"
	"os"

	"github.com/maddsua/syncctl/fsserver/blobstorage"
)

func main() {

	broker := blobstorage.Storage{
		RootDir: "data",
	}

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

	page, err := broker.List("", true, 0, 0)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	for _, entry := range page {
		fmt.Println(">", entry)
	}

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
