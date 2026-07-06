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

func (e *LiveEngine) Cursor() (EngineCursor, error) {
	return e, nil
}

func (e *LiveEngine) Document(id schema.DocumentId, fieldFilter FieldFilter) (Document, error) {
	str, exists := e.idtable.GetStr(id)
	if !exists {
		return Document{}, fmt.Errorf("document with ID %d not found", id)
	}

	doc, _ := e.LoadTreeFields(id)
	doc.Fields = make(map[string]Field)
	doc.Facets = make(map[string][]string)

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

	// TODO fill in Code values where appropriate(?)

	return doc, nil
}

// DOES NOT HANDLE PARENT
func (e *LiveEngine) LoadTreeFields(id schema.DocumentId) (Document, bool) {
	node, exists := e.idtable.GetNode(id)
	if !exists {
		return Document{}, false
	}

	doc := Document{
		Id: id,
	}

	if container := node.Container(); container != nil {
		doc.Container, _ = e.idtable.Get(container.Id())

		for ; container != nil; container = container.Container() {
			if container.Container() == nil {
				doc.RootContainer, _ = e.idtable.Get(container.Id())
			}
		}
	}

	doc.LeafContainee = getLeafContaineeId(node, e.idtable)

	doc.LastDescendant = getLastDescendantId(node, e.idtable)

	if doc.LastDescendant != 0 {
		doc.FirstChild, _ = e.idtable.Get(node.Child(0).Id())
		childCount := node.ChildrenCount()
		doc.LastChild, _ = e.idtable.Get(node.Child(childCount - 1).Id())
	}

	return doc, true
}

func getLeafContaineeId(node *doctree.Node, idtable *doctree.IdTable) schema.DocumentId {
	containees := node.ContainsCount()

	if containees == 0 {
		return 0
	}

	leafContainee := node.Containee(containees - 1)

	for leafContainee.ContainsCount() > 0 {
		leafContainee = leafContainee.Containee(leafContainee.ContainsCount() - 1)
	}

	id, _ := idtable.Get(leafContainee.Id())
	return id
}

func getLastDescendantId(node *doctree.Node, idtable *doctree.IdTable) schema.DocumentId {
	if node.ChildrenCount() == 0 {
		return 0
	}

	lastChild := node.Child(node.ChildrenCount() - 1)

	for lastChild.ChildrenCount() > 0 {
		lastChild = lastChild.Child(lastChild.ChildrenCount() - 1)
	}

	id, _ := idtable.Get(lastChild.Id())
	return id
}

func (e *LiveEngine) Close() {
	// does nothing
}

func (e *LiveEngine) Query(query Query) ([]Document, error) {
	ctx := NewQueryContext(e, query)

	var ids []schema.DocumentId
	var extraData map[schema.DocumentId]map[string]any

	for _, selector := range query.Selectors {
		batch, batchExtraData, err := selector(ctx)
		if err != nil {
			return nil, err
		}

		ids = append(ids, batch...)
		if batchExtraData != nil {
			if extraData == nil {
				extraData = batchExtraData
			} else {
				for id, data := range batchExtraData {
					existingData, exists := extraData[id]
					if !exists {
						extraData[id] = data
					} else {
						for k, v := range data {
							existingData[k] = v
						}
					}
				}
			}
		}
	}

	ids = sortUnique(ids)

	docs := make([]Document, 0, len(ids))

	for _, id := range ids {
		var doc Document
		qdoc, exists := ctx.Documents[id]
		if !exists || !qdoc.Complete {
			fdoc, err := e.Document(id, func(name string, field schema.Field) bool {
				if _, exists := ctx.FilterFields[name]; exists {
					return true
				}

				if _, exists := ctx.ResultFields[name]; exists {
					return true
				}

				return false
			})
			if err != nil {
				return nil, err
			}

			doc = fdoc
		} else {
			doc = qdoc.Document
		}

		doc.Data = extraData[id]

		filtered := false

		for _, filter := range ctx.Filters {
			accept, err := filter.Filter(doc)
			if err != nil {
				return nil, err
			}
			if !accept {
				filtered = true
				break
			}
		}

		if !filtered {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}
