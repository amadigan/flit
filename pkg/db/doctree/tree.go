package doctree

import (
	"fmt"
	"slices"

	"github.com/amadigan/flit/pkg/db"
)

type node struct {
	symbol   symbol
	children []*node
	contains []*node
	id       string
}

type symbol struct {
	str   string
	num   int64
	order int64
}

func (s symbol) String() string {
	if s.str != "" {
		if s.order != 0 {
			return fmt.Sprintf("%s#%d", s.str, s.order)
		}
		return s.str
	}

	if s.order != 0 {
		if s.num != 0 {
			return fmt.Sprintf("%d#%d", s.num, s.order)
		}
		return fmt.Sprintf("#%d", s.order)
	}

	return fmt.Sprintf("%d", s.num)
}

func (s symbol) Equal(other symbol) bool {
	return s.str == other.str && s.num == other.num && s.order == other.order
}

type Tree struct {
	children []*node
}

func (t *Tree) AddChild(symbols []symbol, id string) {
	if len(symbols) == 0 {
		panic("cannot add empty symbol list to tree")
	}

	var child *node

	child, t.children = requireChild(t.children, symbols[0])

	for _, sym := range symbols[1:] {
		child, child.children = requireChild(child.children, sym)
	}

	child.id = id
}

func (t *Tree) AddContainee(symbols []symbol, id string) {
	if len(symbols) == 0 {
		panic("cannot add empty symbol list to tree")
	}

	var child *node

	child, t.children = requireChild(t.children, symbols[0])

	for _, sym := range symbols[1 : len(symbols)-1] {
		contains := false
		for _, c := range child.contains {
			if c.symbol.Equal(sym) {
				child = c
			}
			contains = true
			break
		}

		if !contains {
			child, child.children = requireChild(child.children, sym)
		}
	}

	for i, c := range child.children {
		if c.symbol.Equal(symbols[len(symbols)-1]) {
			// Remove the leaf from children and add it to contains
			child.children = append(child.children[:i], child.children[i+1:]...)
			child.contains = append(child.contains, c)
			break
		}
	}

	child, child.contains = requireChild(child.contains, symbols[len(symbols)-1])
	child.id = id
}

func requireChild(children []*node, sym symbol) (*node, []*node) {
	for _, child := range children {
		if child.symbol.Equal(sym) {
			return child, children
		}
	}

	child := &node{symbol: sym}
	children = append(children, child)
	return child, children
}

func (n *node) sort() {
	slices.SortFunc(n.children, func(a, b *node) int {
		if a.symbol.order != b.symbol.order {
			if a.symbol.order < b.symbol.order {
				return -1
			}
			return 1
		}

		if a.symbol.str != b.symbol.str {
			if a.symbol.str < b.symbol.str {
				return -1
			}
			return 1
		}

		if a.symbol.num != b.symbol.num {
			if a.symbol.num < b.symbol.num {
				return -1
			}
			return 1
		}

		return 0
	})

	for _, child := range n.children {
		child.sort()
	}
}

func (t *Tree) Sort() {
	for _, child := range t.children {
		child.sort()
	}
}

func (t *Tree) IdTable() *IdTable {
	table := &IdTable{
		strToId: make(map[string]db.DocumentId),
	}

	for _, child := range t.children {
		child.collectChildIds(table)
	}

	return table
}

func (n *node) collectContaineeIds(table *IdTable) {
	if len(n.contains) == 0 {
		return
	}

	if n.id != "" {
		table.Add(n.id)
	}

	for _, child := range n.contains {
		child.collectContaineeIds(table)
	}
}

func (n *node) collectChildIds(table *IdTable) {
	if n.id != "" {
		table.Add(n.id)
	}

	n.collectContaineeIds(table)

	for _, child := range n.children {
		child.collectChildIds(table)
	}
}

type IdTable struct {
	strToId map[string]db.DocumentId
	idToStr []string
}

func (t *IdTable) Add(id string) db.DocumentId {
	if _, exists := t.strToId[id]; exists {
		return t.strToId[id]
	}

	t.idToStr = append(t.idToStr, id)
	seq := db.DocumentId(len(t.idToStr))
	if t.strToId == nil {
		t.strToId = make(map[string]db.DocumentId)
	}

	t.strToId[id] = seq
	return seq
}

func (t *IdTable) Get(id string) (db.DocumentId, bool) {
	seq, ok := t.strToId[id]
	return seq, ok
}

func (t *IdTable) GetStr(seq db.DocumentId) (string, bool) {
	point := int(seq) - 1
	if point < 0 || point >= len(t.idToStr) {
		return "", false
	}
	return t.idToStr[point], true
}

func (t *IdTable) Len() int {
	return len(t.idToStr)
}
