package extractor

import "github.com/amadigan/flit/pkg/schema"

type Document interface {
	Fields() DocumentFields
}

type DocumentFields struct {
	Id        string `json:"id"`
	Type      string `json:"type"`
	Parent    string `json:"parent,omitempty"`
	Container string `json:"container,omitempty"`
	Order     int64  `json:"order,omitempty"`
}

func (d DocumentFields) Fields() DocumentFields {
	return d
}

type RootDocument struct {
	DocumentFields
	SourceType string                  `json:"sourceType"`
	Version    string                  `json:"version,omitempty"`
	Language   string                  `json:"lang,omitempty"`
	Code       string                  `json:"code,omitempty"`
	Fields     map[string]schema.Field `json:"fields,omitempty"`
	Types      map[string]schema.Type  `json:"types,omitempty"`
}

type EndDocument struct {
	Id       string              `json:"id"`
	Type     string              `json:"type"`
	Warnings map[string][]string `json:"warnings,omitempty"`
	Errors   map[string][]string `json:"errors,omitempty"`
}

type Ref struct {
	SourceId  string `json:"sourceId"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Length    int    `json:"length,omitempty"`
}
