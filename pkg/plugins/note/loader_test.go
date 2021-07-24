package note

import (
	"path"
	"testing"

	v1 "github.com/byxorna/jot/pkg/types/v1"
)

var (
	fixtures = "../../../test/notes"
)

func TestShortStoragePath(t *testing.T) {
	testcases := map[int64]string{
		1625025715: "2021-06-30.md",
		1625113859: "2021-07-01.md",
	}

	x, err := New(fixtures, false)
	if err != nil {
		t.Fatal(err)
	}

	for input, shortExpectedOutput := range testcases {
		actualOutput := x.StoragePath(v1.ID(input))
		expectedOutput := path.Join(fixtures, shortExpectedOutput)
		if expectedOutput != actualOutput {
			t.Fatalf("Expected %d to yield a storage path of %v but got %v", input, expectedOutput, actualOutput)
		}
	}
}
