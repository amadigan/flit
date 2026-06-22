package extractor

type Document interface {
	Fields() DocumentFields
}

type DocumentFields struct {
	Id       string              `json:"id"`
	Type     string              `json:"type"`
	Title    string              `json:"title,omitempty"`
	Symbol   string              `json:"symbol,omitempty"`
	Path     string              `json:"path,omitempty"`
	FullName []string            `json:"fqn,omitempty"`
	Facets   map[string][]string `json:"facets,omitempty"`
	Content  string              `json:"content,omitempty"`
}

func (d DocumentFields) Fields() DocumentFields {
	return d
}

type RootDocument struct {
	DocumentFields
	SourceType string `json:"sourceType"`
	Version    string `json:"version,omitempty"`
	Language   string `json:"lang,omitempty"`
	Code       string `json:"code,omitempty"`
}

type SourceDocument struct {
	DocumentFields
	Code string `json:"code,omitempty"`
}
