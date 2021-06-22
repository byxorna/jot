// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	te "github.com/muesli/termenv"
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
				m.Entry = n
			} else {
				m.Err = err
			}
		case "up", "j", "left", "h", "p":
			// TODO(gabe): go to more recent entry
			if n, err := m.DB.Next(m.Entry); err != nil {
				m.Entry = n
			} else {
				m.Err = err
			}
		}

	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}

func (m Model) View() string {
	if m.Err != nil {
		return errorView(m.Err, true)
	}
	if m.Entry == nil {
		return errorView(fmt.Errorf("no entry loaded"), false)
	}

	// TODO: switch on state
	r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithEmoji(), glamour.WithEnvironmentConfig(), glamour.WithWordWrap(0))
	md, err := r.Render(m.Entry.Content)
	if err != nil {
		m.Err = err

		return errorView(err, true)
	}

	footerLeft := fmt.Sprintf(" \"%s\" ", m.DB.StoragePath(m.Entry))
	footerRight := fmt.Sprintf(" %3.f%% ", m.viewport.ScrollPercent()*100)
	footerGap := m.viewport.Width - (runewidth.StringWidth(footerLeft) + runewidth.StringWidth(footerRight))
	if footerGap < 0 {
		footerGap = 0
	}
	footer := footerLeft + strings.Repeat(" ", footerGap) + footerRight

	{
		w := lipgloss.Width

		statusKey := statusStyle.Render(fmt.Sprintf("%s", m.DB.Status()))
		encoding := encodingStyle.Render(m.DB.StoragePath(m.Entry))
		scrollPct := fishCakeStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
		// ("🦄 ")
		statusVal := statusText.Copy().
			Width(width - w(statusKey) - w(encoding) - w(scrollPct)).
			Render("")

		bar := lipgloss.JoinHorizontal(lipgloss.Top,
			statusKey,
			statusVal,
			encoding,
			scrollPct,
		)

		footer = statusBarStyle.Width(width).Render(bar)
	}

	dt := fmt.Sprintf(" %s (%d notes)", m.DB.Status(), m.DB.Count())
	headerGap := m.viewport.Width - runewidth.StringWidth(dt)
	if headerGap < 0 {
		headerGap = 0
	}
	header := strings.Repeat(" ", headerGap) + dt

	return lipgloss.JoinVertical(lipgloss.Top, header, md, footer)
}

func errorView(err error, fatal bool) string {
	exitMsg := "press any key to "
	if fatal {
		exitMsg += "exit"
	} else {
		exitMsg += "return"
	}
	s := fmt.Sprintf("%s\n\n%v\n\n%s",
		te.String(" ERROR ").
			Foreground(lib.Cream.Color()).
			Background(lib.Red.Color()).
			String(),
		err,
		common.Subtle(exitMsg),
	)
	return "\n" + indent(s, 3)
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
}
