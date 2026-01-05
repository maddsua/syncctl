package main

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
)

func main() {

	/* if err := create(); err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	} */

	if err := read(); err != nil {
		fmt.Println("ERR", err)
		os.Exit(1)
	}
}

func create() error {

	data := "myshittyshahash"

	file, err := os.Create("data.tar")
	if err != nil {
		return err
	}

	defer file.Close()

	t := tar.NewWriter(file)

	defer t.Flush()

	if err := t.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "hash",
		Size:     int64(len(data)),
		Mode:     int64(fs.ModePerm),
		Format:   tar.FormatGNU,
	}); err != nil {
		return err
	}

	if _, err := t.Write([]byte(data)); err != nil {
		return err
	}

	if err := t.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "data",
		Size:     int64(len(data)),
		Mode:     int64(fs.ModePerm),
		Format:   tar.FormatGNU,
	}); err != nil {
		return err
	}

	if _, err := t.Write([]byte(data)); err != nil {
		return err
	}

	return nil
}

func read() error {

	file, err := os.Open("data.tar")
	if err != nil {
		return err
	}

	defer file.Close()

	t := tar.NewReader(file)

	for {

		entry, err := t.Next()
		if err == io.EOF {
			fmt.Println("EOF")
			break
		} else if err != nil {
			return err
		}

		fmt.Println("ENTRY", entry.Name, entry.Typeflag, entry.Mode, entry.Size)

		data, err := io.ReadAll(t)
		if err != nil {
			return err
		}

		fmt.Println("DATA", string(data))
	}

	return nil
}
