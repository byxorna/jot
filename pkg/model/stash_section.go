// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/stash.go
package model

import (
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	"github.com/charmbracelet/bubbles/paginator"
)

func newSectionModel(name string, be db.DocBackend) section {
	return section{
		name:       name,
		paginator:  newStashPaginator(),
		DocBackend: be,
	}
}

// section contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type section struct {
	// DocBackend is the interface for how we lookup all the documents in this section
	db.DocBackend
	name      string
	paginator paginator.Model
	cursor    int
}

func (s *section) Identifier() string { return s.name }

// TabTitle renders a section into a string for the section tab title
func (s *section) TabTitle(focused bool) string {
	if s.DocBackend == nil {
		return s.name
	}

	itemType := s.DocBackend.DocType().String()
	total := s.DocBackend.Count()
	if total > 1 {
		itemType += "s"
	}
	return fmt.Sprintf("%s: %d %s", s.name, total, itemType)
}
