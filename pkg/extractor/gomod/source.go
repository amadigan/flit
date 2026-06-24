package gomod

import (
	"fmt"
	"go/build"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/amadigan/flit/pkg/extractor"
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

func (c *Context) EmitSourceDocuments(ms *ModuleSource, ch chan<- extractor.Document) error {
	for _, entry := range ms.ListEntries() {
		if entry.IsDir() || hidden.MatchString(entry.Path()) {
			continue
		}

		path := entry.Path()
		name := entry.Name()

		var doc extractor.Document
		var err error

		if strings.HasSuffix(name, ".go") {
			if doc, err = c.buildGoSourceDocument(ms, entry); err != nil {
				return err
			}
		} else {
			for _, extra := range extraFiles {
				if extra.pattern.MatchString(name) {
					if doc, err = c.buildExtraSourceDocument(ms, entry, path, extra.code); err != nil {
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

func buildParent(ms *ModuleSource, entry ModuleFileEntry) string {
	dir := path.Dir(entry.Path())
	if dir == "." {
		return "package:" + ms.Module().Path + "@" + ms.Module().Version
	} else {
		return "package:" + ms.Module().Path + "@" + ms.Module().Version + "/" + dir
	}
}

func (c *Context) buildGoSourceDocument(ms *ModuleSource, entry ModuleFileEntry) (extractor.Document, error) {
	id := fmt.Sprintf("source:%s@%s/%s", ms.Module().Path, ms.Module().Version, entry.Path())

	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	content := string(bs)

	platforms := []string{}
	for _, ctx := range c.BuildContexts {
		if match, err := matchContext(ctx, ms, entry); err != nil {
			return nil, err
		} else if match {
			platforms = append(platforms, ctx.GOOS+"/"+ctx.GOARCH)
		}
	}

	platforms = ms.platformTable.CollapseNames(platforms)

	doc := &SourceDocument{
		DocumentFields: extractor.DocumentFields{
			Id:     id,
			Type:   "source",
			Parent: buildParent(ms, entry),
		},
		Path:      entry.Path(),
		Symbol:    entry.Name(),
		Content:   content,
		Platforms: platforms,
	}

	return doc, nil
}

func matchContext(ctx build.Context, ms *ModuleSource, entry ModuleFileEntry) (bool, error) {
	ctx.OpenFile = ms.Open

	dir := path.Dir(entry.Path())

	return ctx.MatchFile(dir, entry.Name())
}

func (c *Context) buildExtraSourceDocument(ms *ModuleSource, entry ModuleFileEntry, path, code string) (extractor.Document, error) {
	id := fmt.Sprintf("source:%s@%s/%s", ms.Module().Path, ms.Module().Version, entry.Path())

	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	doc := &SourceDocument{
		DocumentFields: extractor.DocumentFields{
			Id:     id,
			Type:   "source",
			Parent: buildParent(ms, entry),
		},
		Path:      path,
		Symbol:    entry.Name(),
		Content:   string(bs),
		Platforms: []string{"all"},
		Code:      code,
	}

	return doc, nil
}
