package db

import (
	"fmt"
	"time"

	"github.com/byxorna/jot/pkg/types"
	"github.com/byxorna/jot/pkg/types/v1"
)

var (
	ErrNoNoteFound = fmt.Errorf("no note found")
	ErrNoNextNote  = fmt.Errorf("no next note found")
	ErrNoPrevNote  = fmt.Errorf("no previous note found")
)

// DB is the interface any plugin satisfies to provide a backend
// for storing and fetching notes
type DB interface {
	// these are implementation specific to the underlying resource Note
	HasNote(v1.ID) bool
	GetByID(v1.ID, bool) (*v1.Note, error)
	CreateOrUpdateNote(*v1.Note) (*v1.Note, error)
	ListAll() ([]*v1.Note, error)
	Next(v1.ID) (*v1.Note, error)
	Previous(v1.ID) (*v1.Note, error)
	ReconcileID(v1.ID) (*v1.Note, error)

	Validate() error

	DocBackend
}

type DocBackend interface { // fs.Store implements this
	DocType() types.DocType
	List() ([]Doc, error)
	Count() int
	Status() v1.SyncStatus
	Get(id types.DocIdentifier, hardread bool) (Doc, error)
	Reconcile(id types.DocIdentifier) (Doc, error)
	StoragePath() string
	StoragePathDoc(id types.DocIdentifier) string
}

type Doc interface {
	Identifier() types.DocIdentifier
	DocType() types.DocType
	MatchesFilter(string) bool

	// UnformattedContent returns the full text, unprocessed with formatting
	UnformattedContent() string
	Created() time.Time
	Modified() *time.Time

	// Content Pills are used to compose interesting doc elements into a View() without
	// needing to unnecessarily complicate the `db` package with UI/View code
	Title() string
	Summary() string        // "3/5 complete", or SubHeading level context
	ExtraContext() []string // teritary context, below title+summary
	Body() string           // the full context of the doc
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
