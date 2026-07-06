package engine

import "github.com/amadigan/flit/pkg/schema"

type FieldFilter func(string, schema.Field) bool

type EngineCursor interface {
	Document(schema.DocumentId, FieldFilter) (Document, error)
	Close()
}

type Engine interface {
	CountDocuments() int
	MaxDocumentId() schema.DocumentId
	Cursor() (EngineCursor, error)
}
