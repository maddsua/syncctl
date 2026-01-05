package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/maddsua/syncctl/fsserver/fs_io"
)

func main() {

	broker := fs_io.FsBroker{
		RootDir: "data",
	}

	/* file, err := broker.Put(context.Background(), &fsserver.FileUpload{
		FileMetaEntry: fsserver.FileMetaEntry{
			Name: "/application-shit/some-lame-file-2.txt",
			Date: time.Unix(5000, 0),
			Size: 6,
		},
		Reader: bytes.NewReader([]byte("test 2")),
	}, true)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	fmt.Println("FILE", file) */

	/* page, err := broker.List(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)

	if err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}

	for _, entry := range page.Entries {
		fmt.Println(">", entry)
	} */

	file, err := broker.Get(context.Background(), "/document.md")
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

	fmt.Println("TEXT", string(text))
}
