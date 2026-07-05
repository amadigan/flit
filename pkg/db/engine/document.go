package engine

import (
	"github.com/amadigan/flit/pkg/schema"
)

type Document struct {
	Id             schema.DocumentId
	Parent         schema.DocumentId
	Container      schema.DocumentId
	RootContainer  schema.DocumentId
	LeafContainee  schema.DocumentId
	FirstChild     schema.DocumentId
	LastChild      schema.DocumentId
	LastDescendant schema.DocumentId
	Symbol         string
	Type           string
	Fields         map[string]Field
	Facets         map[string][]string
}

type Field struct {
	Content   []Text
	FieldType schema.FieldType
	Code      string
	Location  *schema.Ref
}

type Text struct {
	Content []string
	Code    string
}
