package app

import (
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	//"github.com/charmbracelet/bubbles/paginator"
)

// Plugin contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type Plugin struct {
	// DocBackend is the interface for how we lookup all the documents in this section
	db.DocBackend
	name string
}

func (s *Plugin) Identifier() string { return s.name }

func (s Plugin) FilterValue() string {
	return s.Description()
}

func (s *Plugin) Description() string {
	return "desc: " + s.Title()
}

func (s *Plugin) Title() string {
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
