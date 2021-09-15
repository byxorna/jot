package app

import (
	"strings"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
			Padding(0, 0, 0, 2)
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

	activePlugin string

	plugins map[string]Plugin
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
		appStatusBarHeight := 3
		topGap, rightGap, bottomGap, leftGap := appStyle.GetPadding()
		pluginWidth := msg.Width - leftGap - rightGap
		pluginHeight := msg.Height - topGap - appStatusBarHeight - bottomGap
		m.list.SetSize(pluginWidth, pluginHeight)

		for _, pg := range m.plugins {
			pg.SetSize(pluginWidth, pluginHeight)
		}

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		// pass events down to the focused plugin if we have not handled them already
		for _, pg := range m.plugins {
			if pg.Name() == m.activePlugin {
				_, cmd := pg.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

		switch {
		case key.Matches(msg, m.keys.PopStack):
			m.activePlugin = ""
			location := "home"
			return m, m.list.NewStatusMessage(statusMessageStyle(location + " <-"))

		case key.Matches(msg, m.keys.Select):
			m.activePlugin = m.list.SelectedItem().FilterValue()
			return m, m.list.NewStatusMessage(statusMessageStyle("-> " + m.list.SelectedItem().FilterValue()))

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

		default:

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
	stack := []string{"home"}
	if m.activePlugin != "" {
		stack = append(stack, m.activePlugin)
	}
	status := statusStyle.Render(strings.Join(stack, " > "))

	view := ""
	for _, pg := range m.plugins {
		if pg.Name() == m.activePlugin {
			view = pg.View()
			break
		}
	}
	if view == "" {
		view = m.list.View()
	}
	return appStyle.Render(status + "\n" + view)

}
