package db

import (
	"github.com/byxorna/jot/pkg/types/v1"
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
	Count() int

	// TODO: make better methods for finding the "next" entry given a current one
	// TODO: these method names suck, fix this

	Status() v1.SyncStatus
	Validate() error
}
