package gomod

import "github.com/amadigan/flit/pkg/schema"

func GetGoSchema() schema.Schema {
	return schema.Schema{
		Types: map[string]schema.Type{
			"source": {
				Name:       "source",
				ListLabel:  "Source Files",
				FieldOrder: []string{"path", "content"},
			},
			"package": {
				Name:         "package",
				ListLabel:    "Directories",
				FieldOrder:   []string{"path", "symbol", "doc"},
				ContainOrder: [][]string{{"example"}, {"const"}, {"var"}, {"func"}, {"type"}},
				ChildOrder:   [][]string{{"source"}, {"package"}},
			},
			"example": {
				Name:       "example",
				ListLabel:  "Examples",
				FieldOrder: []string{"file", "suffix", "output"},
			},
			"const": {
				Name:       "const",
				ListLabel:  "Constants",
				FieldOrder: []string{"name", "value", "doc"},
			},
			"var": {
				Name:       "var",
				ListLabel:  "Variables",
				FieldOrder: []string{"name", "value", "doc"},
			},
			"type": {
				Name:         "type",
				ListLabel:    "Types",
				FieldOrder:   []string{"name", "doc"},
				ContainOrder: [][]string{{"example"}, {"const"}, {"var"}, {"func"}, {"type"}},
				ChildOrder:   [][]string{{"source"}, {"package"}},
			},
			"func": {
				Name:       "func",
				ListLabel:  "Functions",
				FieldOrder: []string{"name", "signature", "doc"},
			},
		},
		Fields: map[string]schema.Field{
			"path": {
				Label: "Path",
				Type:  schema.FieldTypeString,
			},
			"content": {
				Label: "Content",
				Type:  schema.FieldTypeCode,
				Code:  "go",
			},
			"symbol": {
				Label: "Symbol",
				Type:  schema.FieldTypeSymbol,
			},
			"doc": {
				Label: "Documentation",
				Type:  schema.FieldTypeText,
			},
			"file": {
				Label: "File",
				Type:  schema.FieldTypeCode,
				Code:  "go",
			},
			"suffix": {
				Label: "Suffix",
				Type:  schema.FieldTypeString,
			},
			"output": {
				Label: "Output",
				Type:  schema.FieldTypeCode,
				Code:  "text",
			},
		},
	}
}
