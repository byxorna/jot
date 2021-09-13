package filecommander

import (
	"io/fs"
)

type item struct {
	entryName string
	fs.DirEntry
}

func (i item) FilterValue() string {
	if i.DirEntry != nil {
		return i.DirEntry.Name()
	}
	return i.entryName
}
