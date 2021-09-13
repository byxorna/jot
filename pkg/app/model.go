package app

import (
	"github.com/byxorna/jot/pkg/config"
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
	// TODO(gabe): reenable when its clear how to merge the help in the List with a global help
	//help         help.Model
	lastKey  string
	quitting bool

	plugins []Plugin
	list    list.Model
}

func (m Application) Init() tea.Cmd {
	if m.UseAltScreen {
		return tea.EnterAltScreen
	}
	return nil
}

func (m Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can it can gracefully truncate
		// its view as needed.
		//m.help.Width = msg.Width
		topGap, rightGap, bottomGap, leftGap := appStyle.GetPadding()
		m.list.SetSize(msg.Width-leftGap-rightGap, msg.Height-topGap-bottomGap)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleSpinner):
			cmd := m.list.ToggleSpinner()
			return m, cmd

		case key.Matches(msg, m.keys.ToggleTitleBar):
			v := !m.list.ShowTitle()
			m.list.SetShowTitle(v)
			m.list.SetShowFilter(v)
			m.list.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.list.SetShowStatusBar(!m.list.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil

		}
	}

	newlist, cmd := m.list.Update(msg)
	m.list = newlist
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Application) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	//helpView := m.help.View(m.keys)
	return appStyle.Render(m.list.View()) // + "\n" + helpView)
}
