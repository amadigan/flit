package gomod

import (
	"archive/zip"
	"io"
	"regexp"
	"strings"

	"github.com/amadigan/flit/pkg/extractor"
	"github.com/google/uuid"
)

var extraFiles = []struct {
	pattern *regexp.Regexp
	code    string
}{
	{regexp.MustCompile(`.*\.md$`), "markdown"},
	{regexp.MustCompile(`.*\.txt$`), "text"},
	{regexp.MustCompile(`.*\.yaml$`), "yaml"},
	{regexp.MustCompile(`.*\.yml$`), "yaml"},
	{regexp.MustCompile(`.*\.json$`), "json"},
	{regexp.MustCompile(`.*\.xml$`), "xml"},
	{regexp.MustCompile(`go\.mod$`), "gomod"},
	{regexp.MustCompile(`go\.sum$`), "gosum"},
	{regexp.MustCompile(`^LICENSE(?:\..*)?$`), "license"},
	{regexp.MustCompile(`^COPYING(?:\..*)?$`), "license"},
}

var hidden = regexp.MustCompile(`(^|/)\.[^/]+`)

func EmitSourceDocuments(zr *zip.Reader, ch chan<- extractor.Document) error {
	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() || hidden.MatchString(entry.Name) {
			continue
		}

		path := entry.Name
		name := entry.FileInfo().Name()

		var doc extractor.Document
		var err error

		if strings.HasSuffix(name, ".go") {
			if doc, err = buildGoSourceDocument(entry, path); err != nil {
				return err
			}
		} else {
			for _, extra := range extraFiles {
				if extra.pattern.MatchString(name) {
					if doc, err = buildExtraSourceDocument(entry, path, extra.code); err != nil {
						return err
					}
					break
				}
			}
		}

		if doc != nil {
			ch <- doc
		}
	}
	return nil
}

func buildGoSourceDocument(entry *zip.File, path string) (extractor.Document, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	doc := &extractor.SourceDocument{
		DocumentFields: extractor.DocumentFields{
			Id:      id.String(),
			Type:    "source",
			Path:    path,
			Content: string(bs),
		},
		Code: "go",
	}

	return doc, nil
}

func buildExtraSourceDocument(entry *zip.File, path, code string) (extractor.Document, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	doc := &extractor.SourceDocument{
		DocumentFields: extractor.DocumentFields{
			Id:      id.String(),
			Type:    "source",
			Path:    path,
			Content: string(bs),
		},
		Code: code,
	}

	return doc, nil
}
