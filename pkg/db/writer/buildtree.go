package writer

import (
	"fmt"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/db/doctree"
)

type treeBuilder struct {
	tree      *doctree.Tree
	symbolMap map[string][]doctree.Symbol
	rdb       *ReadWriteDB
	dbinfo    *db.DBInfo
}

func BuildTree(rdb *ReadWriteDB, dbinfo *db.DBInfo) (*doctree.Tree, []string, error) {
	builder := &treeBuilder{
		tree:      &doctree.Tree{},
		symbolMap: make(map[string][]doctree.Symbol),
		rdb:       rdb,
		dbinfo:    dbinfo,
	}

	var roots []string

	for _, id := range rdb.GetDocumentIds() {
		if syms, err := builder.add(id); err != nil {
			return nil, roots, fmt.Errorf("failed to add document %s to tree: %w", id, err)
		} else if syms[len(syms)-1].Type == "root" {
			roots = append(roots, id)
		}
	}

	return builder.tree, roots, nil
}

type treeDoc struct {
	ParentId  string `bdoc:"parent"`
	Container string `bdoc:"container"`
	Type      string `bdoc:"type"`
	Order     int64  `bdoc:"order"`
	Symbol    string `bdoc:"symbol"`
}

func (b *treeBuilder) add(id string) ([]doctree.Symbol, error) {
	symbols, ok := b.symbolMap[id]
	if ok {
		return symbols, nil
	}

	doc := treeDoc{}

	if err := b.rdb.UnmarshalDocument(b.dbinfo, id, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document %s: %w", id, err)
	}

	symbol := doctree.Symbol{
		Name:  doc.Symbol,
		Type:  doc.Type,
		Order: doc.Order,
	}

	var parentSymbols []doctree.Symbol
	if doc.ParentId != "" {
		var err error
		parentSymbols, err = b.add(doc.ParentId)
		if err != nil {
			return nil, fmt.Errorf("failed to add parent document %s: %w", doc.ParentId, err)
		}
	}

	symbols = append(parentSymbols, symbol)
	b.symbolMap[id] = symbols

	if doc.Container != "" {
		symbols, err := b.add(doc.Container)
		if err != nil {
			return nil, fmt.Errorf("failed to add container document %s: %w", doc.Container, err)
		}

		symbols = append(symbols, symbol)

		b.tree.AddContainee(symbols, id)
	} else {
		b.tree.AddChild(symbols, id)
	}

	return symbols, nil
}
