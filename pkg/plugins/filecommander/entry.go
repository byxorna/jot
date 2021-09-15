package filecommander

import (
	"fmt"
	"io/fs"
)

type item struct {
	entryName string
	fs.DirEntry
}

func (i item) Title() string { return i.entryName }
func (i item) Description() string {
	if i.DirEntry == nil {
		return ""
	}

	return fmt.Sprintf("%s", i.DirEntry.Type().Perm().String())
}
func (i item) FilterValue() string {
	if i.DirEntry != nil {
		return i.DirEntry.Name()
	}
	return i.entryName
}
