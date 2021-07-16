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
	Get(id string, hardread bool) (Doc, error)
	Reconcile(id string) (Doc, error)
	StoragePath() string
}

type Doc interface {
	Identifier() string
	//	DocType() types.DocType
	MatchesFilter(string) bool

	// UnformattedContent returns the full text, unprocessed with formatting
	UnformattedContent() string
	Title() string
	Created() time.Time
	Modified() *time.Time

	Validate() error
	SelectorTags() []string
	SelectorLabels() map[string]string
}
