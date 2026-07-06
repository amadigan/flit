package engine

type Query struct {
	Fields  []string
	Filters []Filter
}

type Filter interface {
	RequiredFields() []string
	Filter(Document) (bool, error)
}
