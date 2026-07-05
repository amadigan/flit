package doctree

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amadigan/flit/pkg/schema"
)

type Node struct {
	symbol   Symbol
	children []*Node
	contains []*Node
	id       string
}

type Symbol struct {
	Name  string
	Type  string
	Order int64
}

func (s Symbol) String() string {
	if s.Order != 0 {
		return fmt.Sprintf("%s#%d", s.Name, s.Order)
	}

	return s.Name
}

func (s Symbol) Equal(other Symbol) bool {
	return s.Name == other.Name && s.Type == other.Type && s.Order == other.Order
}

type Tree struct {
	children []*Node
}

func (t *Tree) AddChild(symbols []Symbol, id string) {
	if len(symbols) == 0 {
		panic("cannot add empty symbol list to tree")
	}

	var child *Node

	child, t.children = requireChild(t.children, symbols[0])

	for _, sym := range symbols[1:] {
		child, child.children = requireChild(child.children, sym)
	}

	child.id = id
}

func (t *Tree) AddContainee(symbols []Symbol, id string) {
	if len(symbols) == 0 {
		panic("cannot add empty symbol list to tree")
	}

	var child *Node

	child, t.children = requireChild(t.children, symbols[0])

	for _, sym := range symbols[1 : len(symbols)-1] {
		contains := false
		for _, c := range child.contains {
			if c.symbol.Equal(sym) {
				child = c
				contains = true
				break
			}
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

func (t *Tree) FindNode(symbols []Symbol) *Node {
	if len(symbols) == 0 {
		return nil
	}

	var child *Node

	for _, c := range t.children {
		if c.symbol.Equal(symbols[0]) {
			child = c
			break
		}
	}

	if child == nil {
		return nil
	}

	for _, sym := range symbols[1:] {
		found := false
		for _, c := range child.contains {
			if c.symbol.Equal(sym) {
				child = c
				found = true
				break
			}
		}

		for _, c := range child.children {
			if c.symbol.Equal(sym) {
				child = c
				found = true
				break
			}
		}

		if !found {
			return nil
		}
	}

	return child
}

func requireChild(children []*Node, sym Symbol) (*Node, []*Node) {
	for _, child := range children {
		if child.symbol.Equal(sym) {
			return child, children
		}
	}

	child := &Node{symbol: sym}
	children = append(children, child)
	return child, children
}

func compareNodes(a, b *Node) int {
	if a.symbol.Order != b.symbol.Order {
		if a.symbol.Order < b.symbol.Order {
			return -1
		}
		return 1
	}

	if a.symbol.Name != b.symbol.Name {
		if a.symbol.Name < b.symbol.Name {
			return -1
		}
		return 1
	}

	return 0
}

func (n *Node) sort(sorter schemaSorter) {
	slices.SortFunc(n.contains, sorter.getComparator(n.symbol.Type).compareContains)

	for _, child := range n.contains {
		child.sort(sorter)
	}

	slices.SortFunc(n.children, sorter.getComparator(n.symbol.Type).compareChildren)

	for _, child := range n.children {
		child.sort(sorter)
	}
}

func (t *Tree) Sort(schema schema.Schema) {
	schemaSorter := newSchemaSorter(schema)

	for _, child := range t.children {
		child.sort(schemaSorter)
	}
}

func (t *Tree) IdTable() *IdTable {
	table := &IdTable{
		strToId: make(map[string]schema.DocumentId),
	}

	for _, child := range t.children {
		child.collectChildIds(table)
	}

	return table
}

func (n *Node) collectContaineeIds(table *IdTable) {
	if len(n.contains) == 0 {
		return
	}

	for _, child := range n.contains {
		if child.id != "" {
			table.Add(child)
		}
		child.collectContaineeIds(table)
	}
}

func (n *Node) collectChildIds(table *IdTable) {
	if n.id != "" {
		table.Add(n)
	}

	n.collectContaineeIds(table)

	for _, child := range n.children {
		child.collectChildIds(table)
	}
}

func (t *Tree) String() string {
	var builder strings.Builder

	node := &Node{symbol: Symbol{Name: "root"}, children: t.children}

	for len(node.children) == 1 && len(node.contains) == 0 {
		node = node.children[0]
	}

	builder.WriteString(node.symbol.String())
	if node.id != "" {
		fmt.Fprintf(&builder, " (id: %s, type: %s)", node.id, node.symbol.Type)
	}
	builder.WriteString("\n")

	for _, child := range node.contains {
		child.writeString(&builder, true, 0)
	}

	for _, child := range node.children {
		child.writeString(&builder, false, 0)
	}

	return builder.String()
}

func (n *Node) writeString(builder *strings.Builder, contained bool, depth int) {
	builder.WriteString(strings.Repeat("  ", depth))
	if contained {
		builder.WriteString("-> ")
	} else {
		builder.WriteString("- ")
	}
	builder.WriteString(n.symbol.String())
	if n.id != "" {
		fmt.Fprintf(builder, " (id: %s, type: %s)", n.id, n.symbol.Type)
	}
	builder.WriteString("\n")

	for _, child := range n.contains {
		child.writeString(builder, true, depth+1)
	}

	for _, child := range n.children {
		child.writeString(builder, false, depth+1)
	}
}

type IdTable struct {
	strToId  map[string]schema.DocumentId
	idToNode []*Node
}

func (t *IdTable) Add(node *Node) schema.DocumentId {
	if _, exists := t.strToId[node.id]; exists {
		return t.strToId[node.id]
	}

	t.idToNode = append(t.idToNode, node)
	seq := schema.DocumentId(len(t.idToNode))
	if t.strToId == nil {
		t.strToId = make(map[string]schema.DocumentId)
	}

	t.strToId[node.id] = seq
	return seq
}

func (t *IdTable) Get(id string) (schema.DocumentId, bool) {
	seq, ok := t.strToId[id]
	return seq, ok
}

func (t *IdTable) GetStr(seq schema.DocumentId) (string, bool) {
	point := int(seq) - 1
	if point < 0 || point >= len(t.idToNode) {
		return "", false
	}
	return t.idToNode[point].id, true
}

func (t *IdTable) GetNode(seq schema.DocumentId) (*Node, bool) {
	point := int(seq) - 1
	if point < 0 || point >= len(t.idToNode) {
		return nil, false
	}
	return t.idToNode[point], true
}

func (t *IdTable) GetNodeById(id string) (*Node, bool) {
	seq, ok := t.strToId[id]
	if !ok {
		return nil, false
	}
	return t.GetNode(seq)
}

func (t *IdTable) Len() int {
	return len(t.idToNode)
}

func (t *IdTable) MaxId() schema.DocumentId {
	return schema.DocumentId(len(t.idToNode))
}

type schemaSorter struct {
	children map[string]map[string]int
	contains map[string]map[string]int
}

func newSchemaSorter(schema schema.Schema) schemaSorter {
	children := make(map[string]map[string]int)
	contains := make(map[string]map[string]int)

	for _, typ := range schema.Types {
		containOrder := make(map[string]int)
		childOrder := make(map[string]int)
		children[typ.Name] = childOrder
		contains[typ.Name] = containOrder

		for i, contain := range typ.ContainOrder {
			for _, name := range contain {
				containOrder[name] = i
			}
		}

		for i, child := range typ.ChildOrder {
			for _, name := range child {
				if _, exists := containOrder[name]; !exists {
					containOrder[name] = len(typ.ContainOrder) + i
				}

				childOrder[name] = i
			}
		}
	}

	return schemaSorter{
		children: children,
		contains: contains,
	}
}

type nodeComparator struct {
	contains map[string]int
	children map[string]int
}

func (c nodeComparator) compareContains(a, b *Node) int {
	aOrder, aExists := c.contains[a.symbol.Type]
	bOrder, bExists := c.contains[b.symbol.Type]

	if !aExists {
		aOrder = len(c.contains)
	}

	if !bExists {
		bOrder = len(c.contains)
	}

	if aOrder != bOrder {
		if aOrder < bOrder {
			return -1
		}
		return 1
	}

	return compareNodes(a, b)
}

func (c nodeComparator) compareChildren(a, b *Node) int {
	aOrder, aExists := c.children[a.symbol.Type]
	bOrder, bExists := c.children[b.symbol.Type]

	if !aExists {
		aOrder = len(c.children)
	}

	if !bExists {
		bOrder = len(c.children)
	}

	if aOrder != bOrder {
		if aOrder < bOrder {
			return -1
		}
		return 1
	}

	return compareNodes(a, b)
}

func (s schemaSorter) getComparator(typ string) nodeComparator {
	return nodeComparator{
		contains: s.contains[typ],
		children: s.children[typ],
	}
}

func (n Node) Id() string {
	return n.id
}

func (n Node) Symbol() Symbol {
	return n.symbol
}

func (n Node) ChildrenCount() int {
	return len(n.children)
}

func (n Node) ContainsCount() int {
	return len(n.contains)
}

func (n Node) Child(index int) *Node {
	if index < 0 || index >= len(n.children) {
		return nil
	}
	return n.children[index]
}

func (n Node) Containee(index int) *Node {
	if index < 0 || index >= len(n.contains) {
		return nil
	}
	return n.contains[index]
}
