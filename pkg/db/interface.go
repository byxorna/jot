package db

import (
	"fmt"

	"github.com/byxorna/jot/pkg/types/v1"
)

var (
	ErrNoEntryFound = fmt.Errorf("no entry found")
	ErrNoNextEntry  = fmt.Errorf("no next entry found")
	ErrNoPrevEntry  = fmt.Errorf("no previous entry found")
)

// DB is the interface any plugin satisfies to provide a backend
// for storing and fetching notes
type DB interface {
	HasEntry(v1.ID) bool
	Get(v1.ID, bool) (*v1.Entry, error)
	CreateOrUpdateEntry(*v1.Entry) (*v1.Entry, error)
	ListAll() ([]*v1.Entry, error)
	Next(*v1.Entry) (*v1.Entry, error)
	Previous(*v1.Entry) (*v1.Entry, error)
	StoragePath(v1.ID) string
	Count() int

	Reconcile(v1.ID) (*v1.Entry, error)

	// TODO: make better methods for finding the "next" entry given a current one
	// TODO: these method names suck, fix this

	Status() v1.SyncStatus
	Validate() error
}
