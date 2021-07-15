package model

import (
	"github.com/byxorna/jot/pkg/db"
	tea "github.com/charmbracelet/bubbletea"
)

// Stash represents the entire stash of all document types in each
// backend.  stashModel implements this interface
type Stash interface {
	tea.Model

	Sections() []Section
	FocusedSection() Section
	IsFiltering() bool
}

type Section interface { // section implements this
	db.DocBackend
	TabTitle() string
	Identifier() string
}

type UIDoc interface { // stashItem implements this
	db.Doc
	tea.Model

	ViewWithFilter(string) string
}
