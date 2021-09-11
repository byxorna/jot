package app

import (
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	//"github.com/charmbracelet/bubbles/paginator"
)

func newSectionModel(name string, be db.DocBackend) section {
	return section{
		name: name,
		//paginator:  newStashPaginator(),
		DocBackend: be,
	}
}

// section contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type section struct {
	// DocBackend is the interface for how we lookup all the documents in this section
	db.DocBackend
	name string
	//paginator paginator.Model
	cursor int
}

func (s *section) Identifier() string { return s.name }

func (s *section) FilterValue() string {
	return s.TabTitle()
}

func (s *section) TabTitle() string {
	if s.DocBackend == nil {
		return s.name
	}

	itemType := s.DocBackend.DocType().String()
	items, err := s.DocBackend.List()
	if err != nil {
		return fmt.Sprintf("!! %s", s.name)
	}
	t := itemType
	if len(items) > 1 {
		t = t + "s"
	}
	return fmt.Sprintf("%d %s", len(items), t)
}
