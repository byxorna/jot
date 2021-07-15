// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/stash.go
package model

import (
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	"github.com/charmbracelet/bubbles/paginator"
)

// The types of documents we are currently showing to the user.
type SectionID string

const (
	starlogSectionID       SectionID = "starlog"
	filterSectionID        SectionID = "filter"
	tagSectionID           SectionID = "tags"
	calendarTodaySectionID SectionID = "today"
)

// section contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type section struct {
	// DocBackend is the interface for how we lookup all the documents in this section
	db.DocBackend

	id        SectionID
	paginator paginator.Model
	cursor    int
}

//func (s *section) Init() tea.Cmd {
//	return spinner.Tick
//}

func (s *section) ID() SectionID {
	return s.id
}

func (s *section) TabTitle() string {
	if s.DocBackend == nil {
		return string(s.id)
	}

	switch s.id {
	default:
		return fmt.Sprintf("%d %s", len(s.DocBackend.List()), string(s.id))
	}
}
