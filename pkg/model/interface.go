package model

import (
	"github.com/byxorna/jot/pkg/model/document"
	tea "github.com/charmbracelet/bubbletea"
)

// Stash represents the entire stash of all document types in each
// backend
type Stash interface {
	tea.Model

	Sections() []Section
	FocusedSection() Section
	IsFiltering() bool
}

type Section interface {
	DocBackend
	ID() SectionID
	TabTitle() string
}

type DocBackend interface {
	DocTypes() document.DocTypeSet
	Get(string) Doc
	List() []Doc
}

type Doc interface {
	tea.Model

	ID() string
	DocType() document.DocType
	ViewWithFilter(string) string
	MatchFilter(string) bool
}
