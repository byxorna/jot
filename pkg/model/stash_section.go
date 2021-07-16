// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/stash.go
package model

import (
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	"github.com/charmbracelet/bubbles/paginator"
)

func newSectionModel(name string, be db.DocBackend, settings map[string]string) section {
	return section{
		name:       name,
		settings:   settings,
		paginator:  newStashPaginator(),
		DocBackend: be,
	}
}

// section contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type section struct {
	// DocBackend is the interface for how we lookup all the documents in this section
	db.DocBackend

	settings map[string]string
	name     string

	paginator paginator.Model
	cursor    int
}

func (s *section) Identifier() string { return s.name }

func (s *section) TabTitle() string {
	if s.DocBackend == nil {
		return s.name
	}

	itemType := s.DocBackend.DocType().String()
	items, err := s.DocBackend.List()
	if err != nil {
		return fmt.Sprintf("!! %s", s.name)
	}
	return fmt.Sprintf("%d %s", len(items), itemType)
}
