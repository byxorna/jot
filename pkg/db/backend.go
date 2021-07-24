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

type DocBackendRead interface {
	List() ([]Doc, error)
	Count() int
	Get(id types.DocIdentifier, hardread bool) (Doc, error)
	// TODO: remove Reconcile, it is the same as hard get
	//Reconcile(id types.DocIdentifier) (Doc, error)
}

type DocBackendWrite interface {
}

type DocBackend interface { // fs.Store implements this
	DocBackendRead
	DocBackendWrite

	DocType() types.DocType
	Status() v1.SyncStatus
	StoragePath() string
	StoragePathDoc(id types.DocIdentifier) string
}
