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
			Name: "/myfuckingdocs/verysecretshit.md",
			Date: time.Unix(5000, 0),
			Size: 15,
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

	/* file, err := broker.Get(context.Background(), "/document.md")
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	defer file.Close()
	fmt.Println("FILE", file)

	text, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("TEXT", string(text)) */
}
