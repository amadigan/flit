package gomod

import (
	"github.com/amadigan/flit/pkg/extractor"
	"github.com/amadigan/flit/pkg/schema"
)

var exampleType = schema.Type{
	Label:     "Example",
	ListLabel: "Examples",
}

var fileField = schema.Field{
	Label: "File",
	Type:  schema.FieldTypeCode,
	Code:  "go",
}

var suffixField = schema.Field{
	Label: "Suffix",
	Type:  schema.FieldTypeString,
}

var outputField = schema.Field{
	Label: "Output",
	Type:  schema.FieldTypeCode,
	Code:  "text",
}

type SourceDocument struct {
	extractor.DocumentFields
	Path      string   `json:"path"`
	Symbol    string   `json:"symbol"`
	Code      string   `json:"code,omitempty"`
	Content   string   `json:"content,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

type Package struct {
	extractor.DocumentFields
	DefaultFields
	Readme *extractor.Ref `json:"readme,omitempty"`
}

type Example struct {
	extractor.DocumentFields
	Suffix    string   `json:"suffix,omitempty"`
	File      string   `json:"file,omitempty"`
	Output    string   `json:"output,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

type DefaultFields struct {
	Title       string         `json:"title,omitempty"`
	Symbol      string         `json:"symbol,omitempty"`
	Path        string         `json:"path,omitempty"`
	FullName    []string       `json:"fqn,omitempty"`
	Doc         []string       `json:"doc,omitempty"`
	Signature   string         `json:"signature,omitempty"`
	Declaration *extractor.Ref `json:"declaration,omitempty"`
	Platforms   []string       `json:"platforms,omitempty"`
}

type TypeDocument struct {
	extractor.DocumentFields
	DefaultFields
}

type ValueDocument struct {
	extractor.DocumentFields
	DefaultFields
	Symbols []string `json:"symbols,omitempty"`
}

type FunctionDocument struct {
	extractor.DocumentFields
	DefaultFields
	Recv string `json:"recv,omitempty"`
}
