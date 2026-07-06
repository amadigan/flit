package engine

import (
	"slices"

	"github.com/amadigan/flit/pkg/schema"
)

type Query struct {
	Fields    []string
	Filters   []Filter
	Selectors []Selector
}

type Filter interface {
	RequiredFields() []string
	Filter(Document) (bool, error)
}

type QueryContext struct {
	Documents    map[schema.DocumentId]*QueryDocument
	FilterFields map[string]struct{}
	ResultFields map[string]struct{}
	Filters      []Filter
	Engine       Engine
}

type QueryDocument struct {
	Document Document
	Complete bool
}

type Selector func(QueryContext) ([]schema.DocumentId, map[schema.DocumentId]map[string]any, error)

func ByIdSelector(ids []schema.DocumentId) Selector {
	return func(ctx QueryContext) ([]schema.DocumentId, map[schema.DocumentId]map[string]any, error) {
		return ids, nil, nil
	}
}
func ExpandRootContainerSelector(ids []schema.DocumentId) Selector {
	slices.Sort(ids)
	return func(ctx QueryContext) ([]schema.DocumentId, map[schema.DocumentId]map[string]any, error) {
		roots, err := getRootContainerIds(ctx, ids)
		if err != nil {
			return nil, nil, err
		}

		children, err := getContaineeIds(ctx, roots)
		if err != nil {
			return nil, nil, err
		}

		return children, nil, nil
	}
}

func getRootContainerIds(ctx QueryContext, ids []schema.DocumentId) ([]schema.DocumentId, error) {
	cursor, err := ctx.Engine.Cursor()
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	rootContainerIds := make([]schema.DocumentId, 0, len(ids))
	for _, id := range ids {
		doc, exists := ctx.Documents[id]
		if !exists {
			doc, err := cursor.Document(id, func(string, schema.Field) bool { return false })
			if err != nil {
				return nil, err
			}
			ctx.Documents[id] = &QueryDocument{Document: doc, Complete: false}
		}
		rootContainerIds = append(rootContainerIds, doc.Document.RootContainer)
	}

	return sortUnique(rootContainerIds), nil
}

func getContaineeIds(ctx QueryContext, ids []schema.DocumentId) ([]schema.DocumentId, error) {
	cursor, err := ctx.Engine.Cursor()
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	containeeIds := make([]schema.DocumentId, 0, len(ids))
	for _, id := range ids {
		doc, exists := ctx.Documents[id]
		if !exists {
			doc, err := cursor.Document(id, func(string, schema.Field) bool { return false })
			if err != nil {
				return nil, err
			}
			ctx.Documents[id] = &QueryDocument{Document: doc, Complete: false}
		}

		for descendantId := id; descendantId <= doc.Document.LeafContainee; descendantId++ {
			containeeIds = append(containeeIds, descendantId)
		}
	}

	return sortUnique(containeeIds), nil
}

func sortUnique(ids []schema.DocumentId) []schema.DocumentId {
	slices.Sort(ids)
	unique := make([]schema.DocumentId, 0, len(ids))
	for i, id := range ids {
		if i == 0 || id != ids[i-1] {
			unique = append(unique, id)
		}
	}
	return unique
}

func NewQueryContext(engine Engine, query Query) QueryContext {
	filterFields := make(map[string]struct{})
	for _, filter := range query.Filters {
		for _, field := range filter.RequiredFields() {
			filterFields[field] = struct{}{}
		}
	}

	resultFields := make(map[string]struct{})
	for _, field := range query.Fields {
		resultFields[field] = struct{}{}
	}

	return QueryContext{
		Documents:    make(map[schema.DocumentId]*QueryDocument),
		FilterFields: filterFields,
		ResultFields: resultFields,
		Filters:      query.Filters,
		Engine:       engine,
	}
}
