package fs

import (
	"testing"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
)

var (
	minEntries = 3
)

func TestNext(t *testing.T) {
	loader, err := New("../../../test/notes")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := loader.ListAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) <= minEntries {
		t.Fatal("expected at least %d entries in the test cache, but found %d", minEntries, len(entries))
	}

	for _, e := range entries {
		n, err := loader.Next(e)
		if err == db.ErrNoNextEntry {
		}
	}
}
