// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"os"
	"os/exec"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Mode string

var (
	NormalMode Mode = "normal"
	HelpMode   Mode = "help"
	EditMode   Mode = "edit"
)

type Model struct {
	db.DB

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	Entry    *v1.Entry
	Err      error
	Mode     Mode

	viewport viewport.Model
}

type reloadEntryMsg struct{}
type fileWatchMsg struct{}

func (m Model) Init() tea.Cmd {
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	// TODO: improve this to not be so busy
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m Model) EditCurrentEntry() tea.Cmd {
	m.Mode = EditMode
	oldW, oldH := m.viewport.Width, m.viewport.Height

	filename := m.DB.StoragePath(m.Entry)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	m.Err = cmd.Run()

	m.Mode = NormalMode
	return tea.Sequentially(
		reloadEntryCmd(),
		func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} })
}

func reloadEntryCmd() tea.Cmd {
	return func() tea.Msg { return reloadEntryMsg{} }
}
func errCmd(err error) tea.Cmd {
	return func() tea.Msg { return err }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Date = time.Now()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		//	fmt.Printf("resized:%d:%d", msg.Width, msg.Height)
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.Mode = NormalMode
			return m, nil
		case "?":
			m.Mode = HelpMode
			return m, nil
		case "e":
			return m, m.EditCurrentEntry()
		case "up", "k":
			// go to older entry
			n, err := m.DB.Previous(m.Entry)
			if err != nil {
				if err == db.ErrNoPrevEntry {
					// Swallow errors when we are at the newest entry
					return m, nil
				}
				m.Err = err
				return m, nil
			}
			m.Err = err
			m.Entry = n
			return m, nil
		case "down", "j":
			// TODO(gabe): go to more recent entry
			n, err := m.DB.Next(m.Entry)
			if err != nil {
				if err == db.ErrNoNextEntry {
					return m, nil
				}
				m.Err = err
				return m, nil
			}
			m.Err = err
			m.Entry = n
			return m, nil
		}

	case reloadEntryMsg:
		//fmt.Printf("reloading %d...", m.Entry.ID)
		m.Entry, m.Err = m.DB.Get(m.Entry.ID, true)
		//fmt.Printf("%v", m.Err)
		return m, nil
	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}
