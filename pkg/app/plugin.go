package app

import ()

// Plugin contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type Plugin interface {
	Name() string
	Description() string
	FilterValue() string
}

type plugin struct {
	name string
}

func (p plugin) FilterValue() string {
	panic("fuckfv")
	return p.Description()
}

func (p *plugin) Description() string {
	panic("fuckdesc")
	return "desc: " + p.name
}

func (p *plugin) Name() string {
	panic("fuckname")
	return p.name
}
