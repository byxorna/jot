package db

import (
	"github.com/byxorna/jot/pkg/types/v1"
)

// NotesRepository is the interface any plugin satisfies to provide a backend
// for storing and fetching notes
type NotesRepository interface {
	Create([]string, map[string]string) (*v1.Entry, error)
	Update(*v1.Entry) (*v1.Entry, error)
	List() ([]*v1.Entry, error)
	Status() v1.SyncStatus
	Validate() error
}
