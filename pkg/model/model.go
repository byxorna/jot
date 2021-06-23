// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"fmt"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	db.DB

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	Entry    *v1.Entry
	Err      error

	viewport viewport.Model
}

type fileWatchMsg struct{}
type timeTickMsg struct{}

func (m Model) Init() tea.Cmd {
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	// TODO: improve this to not be so busy
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case timeTickMsg:
		m.Date = time.Now()
		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ", "down", "k", "right", "l", "enter", "n":
			// go to older entry
			if n, err := m.DB.Previous(m.Entry); err != nil {
				fmt.Printf("got err: %v\n", err.Error())
				m.Err = err
			} else {
				m.Entry = n
			}
			return m, nil
		case "up", "j", "left", "h", "p":
			// TODO(gabe): go to more recent entry
			if n, err := m.DB.Next(m.Entry); err != nil {
				fmt.Printf("got err: %v\n", err.Error())
				m.Err = err
			} else {
				m.Entry = n
			}
			return m, nil
		}

	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}
