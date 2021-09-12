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
type DONOTUSE_DB interface {
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

type DocBackendRead interface {
	List() ([]Doc, error)
	Count() int
	Get(id types.DocIdentifier, hardread bool) (Doc, error)
	// TODO: remove Reconcile, it is the same as hard get
	//Reconcile(id types.DocIdentifier) (Doc, error)
}

type DocBackend interface { // fs.Store implements this
	DocBackendRead

	DocType() types.DocType
	Status() v1.SyncStatus
	StoragePath() string
	StoragePathDoc(id types.DocIdentifier) string
}
