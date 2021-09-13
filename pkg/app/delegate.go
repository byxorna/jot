// https://github.com/charmbracelet/bubbletea/blob/master/examples/list-fancy/delegate.go
package app

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

func newSectionDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if plugin, ok := m.SelectedItem().(Plugin); ok {
			title = plugin.Name()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			// TODO(gabe): this should be conditional for each Section's registered handlers
			switch {
			case key.Matches(msg, pluginListKeys.choose):
				return m.NewStatusMessage(statusMessageStyle("Open " + title))

				//case key.Matches(msg, pluginListKeys.remove):
				//	index := m.Index()
				//	m.RemoveItem(index)
				//	if len(m.Items()) == 0 {
				//		pluginListKeys.remove.SetEnabled(false)
				//	}
				//	return m.NewStatusMessage(statusMessageStyle("Deleted " + title))
			}
		}

		return nil
	}

	help := []key.Binding{pluginListKeys.choose} //pluginListKeys.remove}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	choose key.Binding
	//remove key.Binding
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.choose,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
		//remove: key.NewBinding(
		//	key.WithKeys("x", "backspace"),
		//	key.WithHelp("x", "delete"),
		//),
	}
}
