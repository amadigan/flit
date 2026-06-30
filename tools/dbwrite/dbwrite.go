package main

import (
	"fmt"
	"os"

	"github.com/amadigan/flit/pkg/db/writer"
	"github.com/google/uuid"
)

func main() {
	id := uuid.Must(uuid.NewRandom())

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	rdb, err := writer.OpenReadWriteDB(file, id)
	if err != nil {
		panic(err)
	}

	// Use rdb as needed
	doc := map[string]string{
		"example": "data",
	}

	fields, err := writer.MarshalDocument(nil, doc)
	if err != nil {
		panic(err)
	}

	data, length, err := writer.BuildObject(nil, fields)
	if err != nil {
		panic(err)
	}

	if err := rdb.WriteDocumentParts("test", data, length); err != nil {
		panic(err)
	}

	if err := rdb.Close(); err != nil {
		panic(err)
	}
}
