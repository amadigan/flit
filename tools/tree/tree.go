package main

import (
	"fmt"
	"os"

	"github.com/amadigan/flit/pkg/db/writer"
	"github.com/amadigan/flit/pkg/extractor"
	"github.com/amadigan/flit/pkg/schema"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]
	rdb, file, err := openDB(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	dbinfo := writer.DefaultDBInfo()

	tree, roots, err := writer.BuildTree(rdb, dbinfo)
	if err != nil {
		panic(err)
	}

	var rootDoc extractor.RootDocument
	if err := rdb.UnmarshalDocument(dbinfo, roots[0], &rootDoc); err != nil {
		panic(err)
	}

	tree.Sort(schema.Schema{
		Fields: rootDoc.Fields,
		Types:  rootDoc.Types,
	})

	fmt.Printf("Tree built successfully. Root document ID: %s\n", roots[0])
	fmt.Printf("Tree structure:\n")
	fmt.Println(tree.String())

	idTable := tree.IdTable()

	fmt.Printf("ID Table:\n")
	for id := schema.DocumentId(1); id <= idTable.MaxId(); id++ {
		str, exists := idTable.GetStr(id)
		if !exists {
			fmt.Printf("ID %d: <not found>\n", id)
		} else {
			fmt.Printf("ID %d: %s\n", id, str)
		}
	}
}

func openDB(filename string) (*writer.ReadWriteDB, *os.File, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, nil, err
	}

	rdb, err := writer.OpenReadWriteDB(file, uuid.New())
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return rdb, file, nil
}
