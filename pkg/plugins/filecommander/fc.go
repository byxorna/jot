package filecommander

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	units "github.com/docker/go-units"
	"github.com/mitchellh/go-homedir"
)

type Plugin struct {
	sync.Mutex

	// state
	directory string
	err       error
	cache     []os.DirEntry
	finfo     fs.FileInfo

	// UI
	list.Model
}

func New(dir string) (*Plugin, error) {
	p := Plugin{directory: "/"}
	err := p.cd(dir)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Plugin) Description() string {
	if p.finfo == nil {
		return "?"
	}
	sz := p.finfo.Size()
	humanSize := units.BytesSize(float64(sz))
	return fmt.Sprintf("%s %s %s", p.directory, p.finfo.Mode().String(), humanSize)
}

func (p *Plugin) Name() string {
	return "Filecommander"
}

func (p *Plugin) FilterValue() string {
	return fmt.Sprintf("%s %s", p.Name(), p.Description())
}

func (p *Plugin) Count() int {
	return len(p.entries(false))
}

func (p *Plugin) entries(refresh bool) []os.DirEntry {
	if refresh || p.cache == nil {
		p.cache, p.err = os.ReadDir(p.directory)
	}
	return p.cache
}

func (p *Plugin) cd(path string) error {
	p.Lock()
	defer p.Unlock()

	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(expandedPath)
	if err != nil {
		return err
	}

	finfo, err := os.Stat(absDir)
	if err != nil {
		return err
	}
	p.finfo = finfo

	if !finfo.IsDir() {
		return fmt.Errorf("%s must be a directory", absDir)
	}

	p.directory = absDir

	// recreate the list model
	items := []list.Item{
		item{entryName: ".."},
	}
	for _, de := range p.entries(true) {
		items = append(items, item{entryName: de.Name(), DirEntry: de})
	}

	// TODO: sort these items! does Info() perform a stat on the file?
	p.Model = list.NewModel(items, list.NewDefaultDelegate(), 0, 0)

	return nil

}
