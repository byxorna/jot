package model

import (
	"github.com/byxorna/jot/pkg/model/document"
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
	DocBackend
	ID() SectionID
	TabTitle() string
}

type DocBackend interface { // fs.Store implements this
	DocTypes() document.DocTypeSet
	List() []Doc
}

type Doc interface { // stashItem implements this
	tea.Model

	ID() string
	DocType() document.DocType
	ViewWithFilter(string) string
	MatchFilter(string) bool
}
