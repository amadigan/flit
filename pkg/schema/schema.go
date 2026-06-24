package schema

import (
	"maps"
)

const (
	FieldTypeSymbol   = "symbol"
	FieldTypeString   = "string"
	FieldTypeText     = "text"
	FieldTypeCode     = "code"
	FieldTypeMarkdown = "markdown"
	FieldTypeFacet    = "facet"
	FieldTypeRef      = "ref"
)

type Field struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type"`
	Code  string `json:"code,omitempty"`
	Index bool   `json:"index,omitempty"`
}

type Type struct {
	Name       string     `json:"name"`
	Label      string     `json:"label,omitempty"`
	ListLabel  string     `json:"listLabel,omitempty"`
	FieldOrder [][]string `json:"fieldOrder,omitempty"`
	TypeOrder  [][]string `json:"typeOrder,omitempty"`
}

type Ref struct {
	SourceId  int32 `json:"sourceId"`
	Line      int   `json:"line,omitempty"`
	Column    int   `json:"column,omitempty"`
	EndLine   int   `json:"endLine,omitempty"`
	EndColumn int   `json:"endColumn,omitempty"`
	Offset    int   `json:"offset,omitempty"`
	Length    int   `json:"length,omitempty"`
}

type CoreFields struct {
	Id        int32  `json:"id"`
	Parent    int32  `json:"parent,omitempty"`
	Container int32  `json:"container,omitempty"`
	Code      string `json:"code,omitempty"`
}

type DefaultFields struct {
	CoreFields
	Title          string   `json:"title,omitempty"`
	Symbol         string   `json:"symbol,omitempty"`
	Path           string   `json:"path,omitempty"`
	FullName       []string `json:"fqn,omitempty"`
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
	"contents": {
		Type: FieldTypeCode,
	},
}

func GetDefaultFields() map[string]Field {
	return maps.Clone(defaultFields)
}
