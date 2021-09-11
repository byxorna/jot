package app

import (
	"fmt"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	UseHighPerformanceRendering = false
)

type Application struct {
	*config.Config

	UseAltScreen bool
	viewport     viewport.Model
	keys         applicationKeyMap
	help         help.Model
	lastKey      string
	quitting     bool

	pluginList list.Model
}

func (m Application) Init() tea.Cmd {
	return nil
}

func (m Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.lastKey = "↑"
		case key.Matches(msg, m.keys.Down):
			m.lastKey = "↓"
		case key.Matches(msg, m.keys.Left):
			m.lastKey = "←"
		case key.Matches(msg, m.keys.Right):
			m.lastKey = "→"
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}
	pl, cmd := m.pluginList.Update(msg)
	m.pluginList = pl
	return m, cmd
}

func (m Application) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	helpView := m.help.View(m.keys)
	//height := 8 - strings.Count(status, "\n") - strings.Count(helpView, "\n")
	status := fmt.Sprintf("%s", m.lastKey)
	return appStyle.Render(status + "\n" + m.pluginList.View() + "\n" + helpView)
}
