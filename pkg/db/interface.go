package db

import (
	"fmt"

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
	HasNote(v1.ID) bool
	Get(v1.ID, bool) (*v1.Note, error)
	CreateOrUpdateNote(*v1.Note) (*v1.Note, error)
	ListAll() ([]*v1.Note, error)
	Next(v1.ID) (*v1.Note, error)
	Previous(v1.ID) (*v1.Note, error)
	StoragePath(v1.ID) string
	Count() int

	Reconcile(v1.ID) (*v1.Note, error)

	// TODO: make better methods for finding the "next" note given a current one
	// TODO: these method names suck, fix this

	Status() v1.SyncStatus
	Validate() error
}

type DocBackend interface { // fs.Store implements this
	DocTypes() types.DocTypeSet
	List() ([]Doc, error)
	Count() int
}

type Doc interface {
	Identifier() string
	DocType() types.DocType
	MatchesFilter(string) bool
	Validate() error
	SelectorTags() []string
	SelectorLabels() map[string]string
}
