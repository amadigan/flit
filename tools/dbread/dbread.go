package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/db/writer"
	"github.com/google/uuid"
)

func main() {
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

	rdb, err := writer.OpenReadWriteDB(file, uuid.UUID{})
	if err != nil {
		panic(err)
	}

	dbinfo := db.DBInfo{IdKey: "id"}

	ids := rdb.GetDocumentIds()

	fmt.Printf("Found %d documents:\n", len(ids))

	for _, id := range ids {
		m := make(map[string]any)
		if err := rdb.UnmarshalDocument(&dbinfo, id, &m); err != nil {
			panic(err)
		}

		jsonData, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsonData))
	}
}
