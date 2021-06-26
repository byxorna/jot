package fs

import (
	"testing"
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

	if len(entries) < minEntries {
		t.Fatalf("expected at least %d entries in the test cache, but found %d", minEntries, len(entries))
	}

	// TODO:
	//for _, e := range entries {
	//	n, err := loader.Next(e)
	//	if err == db.ErrNoNextEntry {
	//	}
	//}
}
