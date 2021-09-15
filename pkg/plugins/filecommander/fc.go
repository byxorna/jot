package filecommander

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	list list.Model
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

func (p *Plugin) Init() tea.Cmd { return nil }
func (p *Plugin) View() string {
	return p.list.View()
}

func (p *Plugin) Name() string {
	return "Filecommander"
}

func (p *Plugin) FilterValue() string {
	return fmt.Sprintf("%s %s", p.Name(), p.Description())
}

func (p *Plugin) SetSize(width, height int) {
	p.list.SetSize(width, height)
}

func (p *Plugin) Count() int {
	return len(p.entries(false))
}

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	l, cmd := p.list.Update(msg)
	p.list = l
	return p, cmd
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
		item{entryName: ".."}, // Open a new finfo
	}
	for _, de := range p.entries(true) {
		de2 := de
		items = append(items, item{entryName: de.Name(), DirEntry: de2})
	}

	// TODO: sort these items! does Info() perform a stat on the file?
	p.list = list.NewModel(items, list.NewDefaultDelegate(), 0, 0)
	p.list.Title = p.directory

	return nil

}
