package gomod

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/amadigan/flit/pkg/extractor"
)

type sourceFile struct {
	path      string
	platforms []string
	astFile   *ast.File
}

func (c *Context) GoDoc(ms *ModuleSource, srcChan <-chan extractor.Document, outChan chan<- extractor.Document) error {
	packages := map[string][]sourceFile{}
	readmes := map[string]*extractor.Ref{}

	for doc := range srcChan {
		srcdoc, ok := doc.(*SourceDocument)
		if !ok {
			continue
		}

		name := path.Base(srcdoc.Path)

		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, "README") {
			readmes[path.Dir(srcdoc.Path)] = &extractor.Ref{
				SourceId: srcdoc.Id,
			}
		}

		if !strings.HasSuffix(srcdoc.Path, ".go") {
			continue
		}

		platforms := srcdoc.Platforms

		if len(platforms) == 0 {
			continue
		}

		dir := path.Dir(srcdoc.Path)
		packages[dir] = append(packages[dir], sourceFile{path: name, platforms: platforms})
	}

	for dir, sources := range packages {
		if err := c.emitPackageDocs(ms, outChan, dir, sources, readmes[dir]); err != nil {
			return fmt.Errorf("failed to emit package docs for %s: %w", dir, err)
		}
	}

	return nil
}

func (c *Context) emitPackageDocs(ms *ModuleSource, outChan chan<- extractor.Document, dir string, sources []sourceFile, readme *extractor.Ref) error {
	fset := token.NewFileSet()
	platforms := map[string]struct{}{}
	filePlatforms := map[string][]string{}

	for i, src := range sources {
		for _, platform := range src.platforms {
			platforms[platform] = struct{}{}
		}

		filePlatforms[src.path] = src.platforms

		bs, err := ms.Open(src.path)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", src.path, err)
		}

		astFile, err := parser.ParseFile(fset, src.path, bs, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse source file %s: %w", src.path, err)
		}

		src.astFile = astFile
		sources[i] = src
	}

	if len(platforms) > 1 {
		delete(platforms, "all")
	}

	platformList := make([]string, 0, len(platforms))
	for platform := range platforms {
		platformList = append(platformList, platform)
	}
	sort.Strings(platformList)

	srcRoot := path.Join(ms.Module().Path, dir)
	wrotePackage := false

	for _, platform := range platformList {
		var astFiles []*ast.File
		for _, src := range sources {
			if slices.Contains(src.platforms, platform) || slices.Contains(src.platforms, "all") {
				astFiles = append(astFiles, src.astFile)
			}
		}

		pkg, err := doc.NewFromFiles(fset, astFiles, srcRoot, doc.AllDecls|doc.AllMethods|doc.PreserveAST)
		if err != nil {
			return fmt.Errorf("failed to create doc package for %s: %w", dir, err)
		}

		pkgDoc := &Package{
			DocumentFields: extractor.DocumentFields{
				Id:   path.Join(fmt.Sprintf("package:%s@%s", ms.Module().Path, ms.Module().Version), dir),
				Type: "package",
			},
			DefaultFields: DefaultFields{
				Symbol:    pkg.ImportPath,
				Platforms: platformList,
			},
			Readme: readme,
		}

		if len(pkg.Doc) > 0 {
			pkgDoc.Doc = []string{string(pkg.Markdown(pkg.Doc))}
		}

		if dir != "." {
			pkgDoc.Parent = path.Join(fmt.Sprintf("package:%s@%s", ms.Module().Path, ms.Module().Version), path.Dir(dir))
		} else {
			pkgDoc.Parent = fmt.Sprintf("go:%s@%s", ms.Module().Path, ms.Module().Version)
		}

		if !wrotePackage {
			outChan <- pkgDoc
			wrotePackage = true
		}

		for _, cnst := range pkg.Consts {
			cnstDoc, err := buildValueDoc(ms, pkg, pkgDoc.Id, fset, filePlatforms, cnst, valueTypeConst)
			if err != nil {
				return fmt.Errorf("failed to build const doc for %s: %w", dir, err)
			}

			if cnstDoc != nil {
				outChan <- cnstDoc
			}
		}

		for _, variable := range pkg.Vars {
			varDoc, err := buildValueDoc(ms, pkg, pkgDoc.Id, fset, filePlatforms, variable, valueTypeVar)
			if err != nil {
				return fmt.Errorf("failed to build var doc for %s: %w", dir, err)
			}

			if varDoc != nil {
				outChan <- varDoc
			}
		}

		for _, fun := range pkg.Funcs {
			if err := crawlFunc(ms, pkg, pkgDoc.Id, fset, filePlatforms, fun, outChan); err != nil {
				return fmt.Errorf("failed to crawl function %s: %w", fun.Name, err)
			}
		}

		for _, typ := range pkg.Types {
			if err := crawlType(ms, pkg, pkgDoc.Id, fset, filePlatforms, typ, outChan); err != nil {
				return fmt.Errorf("failed to crawl type %s: %w", typ.Name, err)
			}
		}

		for _, example := range pkg.Examples {
			exampleDoc, err := buildExampleDoc(pkg, pkgDoc.Id, fset, filePlatforms, example)
			if err != nil {
				return fmt.Errorf("failed to build example doc for %s: %w", dir, err)
			}

			if exampleDoc != nil {
				outChan <- exampleDoc
			}
		}

		for _, astFile := range astFiles {
			delete(filePlatforms, fset.Position(astFile.Pos()).Filename)
		}
	}

	return nil
}

func nodeId(node ast.Node, fset *token.FileSet) string {
	start := fset.Position(node.Pos())
	end := fset.Position(node.End())
	return fmt.Sprintf("%s:%d-%d", start.Filename, start.Offset, end.Offset)
}

func buildExampleDoc(pkg *doc.Package, parent string, fset *token.FileSet, filePlatforms map[string][]string, example *doc.Example) (extractor.Document, error) {
	platforms, ok := filePlatforms[fset.Position(example.Code.Pos()).Filename]
	if !ok {
		return nil, nil // skip
	}

	doc := &Example{
		DocumentFields: extractor.DocumentFields{
			Id:     fmt.Sprintf("example:%s/%s", parent, nodeId(example.Code, fset)),
			Type:   "example",
			Parent: parent,
		},
		Suffix:    example.Suffix,
		Output:    example.Output,
		Platforms: platforms,
	}

	var buf bytes.Buffer
	node := example.Code
	if example.Play != nil {
		node = example.Play
	}

	err := format.Node(&buf, fset, node)
	if err != nil {
		return nil, fmt.Errorf("failed to format example code: %w", err)
	}

	doc.File = buf.String()

	return doc, nil
}

func crawlType(ms *ModuleSource, pkg *doc.Package, parent string, fset *token.FileSet, filePlatforms map[string][]string, typ *doc.Type, ch chan<- extractor.Document) error {
	id := fmt.Sprintf("type:%s/%s/%s", parent, typ.Name, nodeId(typ.Decl, fset))

	pos := fset.Position(typ.Decl.Pos())
	end := fset.Position(typ.Decl.End())

	platforms, ok := filePlatforms[fset.Position(typ.Decl.Pos()).Filename]

	if ok {
		typeDoc := &TypeDocument{
			DocumentFields: extractor.DocumentFields{
				Id:     id,
				Type:   "type",
				Parent: parent,
			},
			DefaultFields: DefaultFields{
				Symbol: typ.Name,
				Declaration: &extractor.Ref{
					SourceId: fmt.Sprintf("source:%s@%s/%s", ms.Module().Path, ms.Module().Version, pos.Filename),
					Offset:   pos.Offset,
					Length:   end.Offset - pos.Offset,
				},
				Platforms: platforms,
			},
		}

		if len(typ.Doc) > 0 {
			typeDoc.Doc = []string{string(pkg.Markdown(typ.Doc))}
		}

		ch <- typeDoc
	}

	for _, cnst := range typ.Consts {
		cnstDoc, err := buildValueDoc(ms, pkg, id, fset, filePlatforms, cnst, valueTypeConst)
		if err != nil {
			return fmt.Errorf("failed to build const doc for type %s: %w", typ.Name, err)
		}

		if cnstDoc != nil {
			ch <- cnstDoc
		}
	}

	for _, variable := range typ.Vars {
		varDoc, err := buildValueDoc(ms, pkg, id, fset, filePlatforms, variable, valueTypeVar)
		if err != nil {
			return fmt.Errorf("failed to build var doc for type %s: %w", typ.Name, err)
		}

		if varDoc != nil {
			ch <- varDoc
		}
	}

	for _, method := range typ.Methods {
		if err := crawlFunc(ms, pkg, id, fset, filePlatforms, method, ch); err != nil {
			return fmt.Errorf("failed to crawl method %s: %w", method.Name, err)
		}
	}

	for _, fun := range typ.Funcs {
		if err := crawlFunc(ms, pkg, id, fset, filePlatforms, fun, ch); err != nil {
			return fmt.Errorf("failed to crawl function %s: %w", fun.Name, err)
		}
	}

	for _, example := range typ.Examples {
		exampleDoc, err := buildExampleDoc(pkg, id, fset, filePlatforms, example)
		if err != nil {
			return fmt.Errorf("failed to build example doc for type %s: %w", typ.Name, err)
		}

		if exampleDoc != nil {
			ch <- exampleDoc
		}
	}

	return nil
}

type valueType string

const (
	valueTypeConst valueType = "const"
	valueTypeVar   valueType = "var"
)

func buildValueDoc(ms *ModuleSource, pkg *doc.Package, parent string, fset *token.FileSet, filePlatforms map[string][]string, value *doc.Value, typ valueType) (extractor.Document, error) {
	pos := fset.Position(value.Decl.Pos())
	end := fset.Position(value.Decl.End())

	platforms, ok := filePlatforms[fset.Position(value.Decl.Pos()).Filename]

	if !ok {
		return nil, nil // skip
	}

	id := fmt.Sprintf("%s:%s@%s/%s", typ, ms.Module().Path, ms.Module().Version, nodeId(value.Decl, fset))

	valueDoc := &ValueDocument{
		DocumentFields: extractor.DocumentFields{
			Id:     id,
			Type:   string(typ),
			Parent: parent,
		},
		DefaultFields: DefaultFields{
			Symbol: value.Names[0],
			Declaration: &extractor.Ref{
				SourceId: fmt.Sprintf("source:%s@%s/%s", ms.Module().Path, ms.Module().Version, pos.Filename),
				Offset:   pos.Offset,
				Length:   end.Offset - pos.Offset,
			},
			Platforms: platforms,
		},
	}

	if len(value.Doc) > 0 {
		valueDoc.Doc = []string{string(pkg.Markdown(value.Doc))}
	}

	return valueDoc, nil
}

func crawlFunc(ms *ModuleSource, pkg *doc.Package, parent string, fset *token.FileSet, filePlatforms map[string][]string, fn *doc.Func, ch chan<- extractor.Document) error {
	pos := fset.Position(fn.Decl.Pos())
	end := fset.Position(fn.Decl.End())

	id := fmt.Sprintf("func:%s/%s/%s", parent, fn.Name, nodeId(fn.Decl, fset))

	platforms, ok := filePlatforms[fset.Position(fn.Decl.Pos()).Filename]

	if ok {
		funcDoc := &FunctionDocument{
			DocumentFields: extractor.DocumentFields{
				Id:     id,
				Type:   "func",
				Parent: parent,
			},
			DefaultFields: DefaultFields{
				Symbol: fn.Name,
				Declaration: &extractor.Ref{
					SourceId: fmt.Sprintf("source:%s@%s/%s", ms.Module().Path, ms.Module().Version, pos.Filename),
					Offset:   pos.Offset,
					Length:   end.Offset - pos.Offset,
				},
				Platforms: platforms,
			},
			Recv: fn.Recv,
		}

		if len(fn.Doc) > 0 {
			funcDoc.Doc = []string{string(pkg.Markdown(fn.Doc))}
		}

		ch <- funcDoc
	}

	for _, example := range fn.Examples {
		exampleDoc, err := buildExampleDoc(pkg, id, fset, filePlatforms, example)
		if err != nil {
			return fmt.Errorf("failed to build example doc for function %s: %w", fn.Name, err)
		}

		if exampleDoc != nil {
			ch <- exampleDoc
		}
	}

	return nil
}
