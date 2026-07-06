package engine

import (
	"net/url"
	"strings"

	"github.com/amadigan/flit/pkg/schema"
)

type PageTree struct {
	Pages map[string]schema.DocumentId
	Roots []*PageTreeNode
}

type PageTreeNode struct {
	Children   map[string]schema.DocumentId
	ChildNodes []*PageTreeNode
}

func JoinFQN(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	var builder strings.Builder

	for i, part := range parts {
		if i > 0 {
			builder.WriteString("/")
		}
		builder.WriteString(url.PathEscape(part))
	}

	return builder.String()
}
