// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/markdown.go
package model

import (
	"github.com/byxorna/jot/pkg/db"
	"github.com/charmbracelet/glamour"
)

var (
	mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithEmoji(),
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(0))
)

// Sort documents with local files first, then by date.
type markdownsByLocalFirst []*stashItem

func (m markdownsByLocalFirst) Len() int      { return len(m) }
func (m markdownsByLocalFirst) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m markdownsByLocalFirst) Less(i, j int) bool {
	// Neither are local files so sort by date descending
	return m[i].Created().After(m[j].Created())
}

func AsStashItem(d db.Doc, backend db.DocBackend) *stashItem {
	i := stashItem{Doc: d, DocBackend: backend}
	return &i
}
