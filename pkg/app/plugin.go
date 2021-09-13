package app

import ()

// Plugin contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type Plugin interface {
	Name() string
	Description() string
	FilterValue() string
	View() string
}

type item struct {
	title, desc string
}

func (i item) Name() string        { return i.Title() }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }
