package main

import (
	"os"

	"github.com/amadigan/flit/internal/util"
	"github.com/amadigan/flit/pkg/extractor"
	"github.com/amadigan/flit/pkg/extractor/gomod"
)

func main() {
	pls, err := gomod.GetSupportedPlatforms()
	if err != nil {
		panic(err)
	}

	pt := gomod.NewPlatformTable(pls)
	buildctx := gomod.NewContext(pls)

	mod, err := gomod.Resolve(os.Args[1], os.Args[2])
	if err != nil {
		panic(err)
	}

	modId := "go:" + mod.Path + "@" + mod.Version

	ms, err := gomod.OpenModuleSource(pt, mod)
	if err != nil {
		panic(err)
	}

	var errs []extractor.WriterError
	errChan := make(chan extractor.WriterError)
	go func() {
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	sourceChan := make(chan extractor.ExtractStream, 16)

	writer := extractor.NewExtractWriter(sourceChan, errChan, os.Stdout)

	root := extractor.RootDocument{
		DocumentFields: extractor.DocumentFields{
			Id:   modId,
			Type: "root",
		},
		SourceType: "go",
		Version:    mod.Version,
		Code:       "go",
	}

	if err := writer.Open(root); err != nil {
		panic(err)
	}

	fileSink := make(chan extractor.Document, 16)
	fileXfer := make(chan extractor.Document, 16)
	fileDoc := make(chan extractor.Document, 16)

	util.Distribute(fileSink, fileXfer, fileDoc)

	sourceChan <- extractor.ExtractStream{
		Source: fileXfer,
	}

	docChan := make(chan extractor.Document, 16)

	sourceChan <- extractor.ExtractStream{
		Source: docChan,
	}

	docErr := make(chan error)

	go func() {
		defer close(docChan)
		defer close(docErr)
		docErr <- buildctx.GoDoc(ms, fileDoc, docChan)
	}()

	if err := buildctx.EmitSourceDocuments(ms, fileSink); err != nil {
		panic(err)
	}

	close(fileSink)

	if err := <-docErr; err != nil {
		panic(err)
	}

	if err := writer.CloseRoot(extractor.EndDocument{Id: modId, Type: "end"}); err != nil {
		panic(err)
	}

	if err := writer.Close(); err != nil {
		panic(err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			if rootErr, ok := err.Cause.(*extractor.RootClosedError); ok {
				if rootErr.RootId == modId {
					continue
				}
			}
			panic(err)
		}
	}
}
