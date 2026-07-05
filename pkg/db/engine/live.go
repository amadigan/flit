package engine

import (
	"fmt"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/db/doctree"
	"github.com/amadigan/flit/pkg/db/writer"
	"github.com/amadigan/flit/pkg/extractor"
	"github.com/amadigan/flit/pkg/schema"
)

type LiveEngine struct {
	rdb     *writer.ReadWriteDB
	dbinfo  *db.DBInfo
	tree    *doctree.Tree
	idtable *doctree.IdTable
	schema  schema.Schema
	root    string
}

func NewLiveEngine(rdb *writer.ReadWriteDB) (*LiveEngine, error) {
	dbinfo := writer.DefaultDBInfo()

	tree, roots, err := writer.BuildTree(rdb, dbinfo)
	if err != nil {
		return nil, err
	}

	if len(roots) != 1 {
		return nil, fmt.Errorf("expected exactly one root document, found %d", len(roots))
	}

	root := roots[0]

	var rootDoc extractor.RootDocument
	if err := rdb.UnmarshalDocument(dbinfo, root, &rootDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal root document %s: %w", root, err)
	}

	tree.Sort(schema.Schema{
		Fields: rootDoc.Fields,
		Types:  rootDoc.Types,
	})

	idtable := tree.IdTable()

	return &LiveEngine{
		rdb:     rdb,
		dbinfo:  dbinfo,
		tree:    tree,
		idtable: idtable,
		schema:  schema.Schema{Fields: rootDoc.Fields, Types: rootDoc.Types},
		root:    root,
	}, nil
}

func (e *LiveEngine) CountDocuments() int {
	return e.rdb.GetDocumentCount()
}

func (e *LiveEngine) MaxDocumentId() schema.DocumentId {
	return e.idtable.MaxId()
}

func (e *LiveEngine) GetDocument(id schema.DocumentId, fieldFilter func(string, schema.Field) bool) (Document, error) {
	str, exists := e.idtable.GetStr(id)
	if !exists {
		return Document{}, fmt.Errorf("document with ID %d not found", id)
	}

	doc := Document{
		Id:     id,
		Fields: make(map[string]Field),
		Facets: make(map[string][]string),
	}

	reader := e.rdb.OpenDocument(str)
	if reader == nil {
		return Document{}, fmt.Errorf("document with ID %d not found", id)
	}

	cursor, headers, err := db.ReadObject(e.dbinfo, reader, 0)
	if err != nil {
		return Document{}, fmt.Errorf("failed to read document with ID %d: %w", id, err)
	}

	for i := range headers {
		name, err := cursor.Name(i)
		if err != nil {
			return Document{}, fmt.Errorf("failed to read field name for document with ID %d: %w", id, err)
		}

		if name == "symbol" {
			doc.Symbol, err = parseString(i, headers[i], cursor)
			if err != nil {
				return Document{}, fmt.Errorf("failed to parse symbol for document with ID %d: %w", id, err)
			}
		} else if name == "type" {
			doc.Type, err = parseString(i, headers[i], cursor)
			if err != nil {
				return Document{}, fmt.Errorf("failed to parse type for document with ID %d: %w", id, err)
			}
			continue
		} else if name == "parent" {
			parentId, err := parseString(i, headers[i], cursor)
			if err != nil {
				return Document{}, fmt.Errorf("failed to parse parent for document with ID %d: %w", id, err)
			}

			doc.Parent, _ = e.idtable.Get(parentId)
			continue
		}

		fieldSchema, exists := e.schema.Fields[name]
		if !exists {
			continue
		}

		if fieldFilter != nil && !fieldFilter(name, fieldSchema) {
			continue
		}

		if fieldSchema.Type == schema.FieldTypeFacet {
			facets, err := parseStringArray(i, cursor)
			if err != nil {
				return Document{}, fmt.Errorf("failed to parse facets for document with ID %d: %w", id, err)
			}
			doc.Facets[name] = facets
		} else {
			field, err := e.parseField(i, headers, cursor)
			if err != nil {
				return Document{}, fmt.Errorf("failed to parse field %s for document with ID %d: %w", name, id, err)
			}

			field.FieldType = fieldSchema.Type
			if field.Code == "" {
				field.Code = fieldSchema.Code
			}

			doc.Fields[name] = field
		}
	}

	// TODO use the doctree to populate descendant, containee, and child fields
	// TODO fill in Code values where appropriate

	return doc, nil
}
