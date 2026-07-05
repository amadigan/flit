package schema

import (
	"maps"
)

type DocumentId int32

type FieldType string

const (
	FieldTypeSymbol   FieldType = "symbol"
	FieldTypeString   FieldType = "string"
	FieldTypeText     FieldType = "text"
	FieldTypeCode     FieldType = "code"
	FieldTypeMarkdown FieldType = "markdown"
	FieldTypeFacet    FieldType = "facet"
	FieldTypeRef      FieldType = "ref"
)

type Schema struct {
	Fields map[string]Field `json:"fields"`
	Types  map[string]Type  `json:"types"`
}

type Field struct {
	Label string    `json:"label,omitempty"`
	Type  FieldType `json:"type"`
	Code  string    `json:"code,omitempty"`
	Index bool      `json:"index,omitempty"`
}

type Type struct {
	Name         string     `json:"name"`
	Label        string     `json:"label,omitempty"`
	ListLabel    string     `json:"listLabel,omitempty"`
	FieldOrder   []string   `json:"fieldOrder,omitempty"`
	ContainOrder [][]string `json:"containOrder,omitempty"`
	ChildOrder   [][]string `json:"childOrder,omitempty"`
}

type Ref struct {
	SourceId  DocumentId `json:"sourceId" bdoc:"sourceId"`
	Line      int        `json:"line,omitempty"`
	Column    int        `json:"column,omitempty"`
	EndLine   int        `json:"endLine,omitempty" bdoc:"endLine"`
	EndColumn int        `json:"endColumn,omitempty" bdoc:"endColumn"`
	Offset    int        `json:"offset,omitempty"`
	Length    int        `json:"length,omitempty"`
}

type CoreFields struct {
	Id        DocumentId `json:"id"`
	Parent    DocumentId `json:"parent,omitempty"`
	Container DocumentId `json:"container,omitempty"`
	Code      string     `json:"code,omitempty"`
}

type DefaultFields struct {
	CoreFields
	Title          string   `json:"title,omitempty"`
	Symbol         string   `json:"symbol,omitempty"`
	Path           string   `json:"path,omitempty"`
	FullName       string   `json:"fqn,omitempty" bdoc:"fqn"`
	Doc            []string `json:"doc,omitempty"`
	Signature      string   `json:"signature,omitempty"`
	Declaration    *Ref     `json:"declaration,omitempty"`
	Implementation *Ref     `json:"implementation,omitempty"`
}

var defaultFields = map[string]Field{
	"title": {
		Label: "Title",
		Type:  FieldTypeString,
	},
	"symbol": {
		Label: "Symbol",
		Type:  FieldTypeSymbol,
	},
	"path": {
		Label: "Path",
		Type:  FieldTypeString,
	},
	"fqn": {
		Label: "Fully Qualified Name",
		Type:  FieldTypeString,
	},
	"doc": {
		Type: FieldTypeMarkdown,
	},
	"signature": {
		Label: "Signature",
		Type:  FieldTypeMarkdown,
	},
	"declaration": {
		Label: "Declaration",
		Type:  FieldTypeRef,
	},
	"implementation": {
		Label: "Implementation",
		Type:  FieldTypeRef,
	},
	"content": {
		Type: FieldTypeCode,
	},
}

func GetDefaultFields() map[string]Field {
	return maps.Clone(defaultFields)
}
