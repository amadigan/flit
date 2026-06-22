package main

import (
	"archive/zip"
	"encoding/json"
	"os"

	"github.com/amadigan/flit/pkg/extractor"
	"github.com/amadigan/flit/pkg/extractor/gomod"
)

func main() {
	mod, err := gomod.Resolve(os.Args[1], os.Args[2])
	if err != nil {
		panic(err)
	}

	f, err := os.Open(mod.Zip)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		panic(err)
	}

	ch := make(chan extractor.Document)
	go func() {
		defer close(ch)
		err = gomod.EmitSourceDocuments(zr, ch)
		if err != nil {
			panic(err)
		}
	}()

	for doc := range ch {
		bs, err := json.Marshal(doc)
		if err != nil {
			panic(err)
		}
		os.Stdout.Write(bs)
		os.Stdout.Write([]byte{'\n'})
	}
}
