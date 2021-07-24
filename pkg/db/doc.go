package db

import (
	"time"

	"github.com/byxorna/jot/pkg/types"
)

type Doc interface {
	Identifier() types.ID
	DocType() types.DocType
	MatchesFilter(string) bool

	Created() time.Time
	Modified() *time.Time
	Trashed() *time.Time

	// Content Pills are used to compose interesting doc elements into a View() without
	// needing to unnecessarily complicate the `db` package with UI/View code
	Title() string
	Summary() string        // "3/5 complete", or SubHeading level context
	ExtraContext() []string // teritary context, below title+summary
	AsMarkdown() string     // the full context of the doc
	Links() map[string]string
	Icon() string

	Validate() error
	SelectorTags() []string
	SelectorLabels() map[string]string
}

type DocsByCreated []Doc

func (m DocsByCreated) Len() int      { return len(m) }
func (m DocsByCreated) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m DocsByCreated) Less(i, j int) bool {
	// Neither are local files so sort by date descending
	return m[i].Created().After(m[j].Created())
}

type DocsByModified []Doc

func (m DocsByModified) Len() int      { return len(m) }
func (m DocsByModified) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m DocsByModified) Less(i, j int) bool {
	// Neither are local files so sort by date descending
	var ti, tj time.Time
	if m[i].Modified() != nil {
		ti = *m[i].Modified()
	} else {
		ti = m[i].Created()
	}
	if m[j].Modified() != nil {
		tj = *m[j].Modified()
	} else {
		tj = m[j].Created()
	}
	return ti.After(tj)
}
