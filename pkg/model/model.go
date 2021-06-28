// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Mode string

var (
	NormalMode Mode = "day view"
	HelpMode   Mode = "help"
	EditMode   Mode = "edit"

	Rapid         = time.Second * 1
	Informational = time.Second * 5
	Forever       = time.Duration(0)
)

type Model struct {
	db.DB

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	Entry    *v1.Entry
	Mode     Mode

	messages []*userMessage
	viewport viewport.Model
}

type userMessage struct {
	// Time is when the message happened
	Time time.Time
	// Message is the terse oneline description of the issue
	Message string
	IsError bool
}

// LogUserNotice registers an informational message with the app for display
// via UI, logs, whatever
func (m *Model) LogUserError(err error) {
	if m.messages == nil {
		m.messages = []*userMessage{}
	}
	m.messages = append(m.messages, &userMessage{
		Time:    time.Now(),
		Message: err.Error(),
		IsError: true,
	})
}

func (m *Model) LogUserNotice(msg string) {
	if m.messages == nil {
		m.messages = []*userMessage{}
	}
	m.messages = append(m.messages, &userMessage{
		Time:    time.Now(),
		Message: msg,
		IsError: false,
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Date = time.Now()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.LogUserNotice("entry view")
			m.Mode = NormalMode
			return m, nil
		case "?":
			m.LogUserNotice("use esc to return")
			m.Mode = HelpMode
			return m, nil
		case "r":
			m.LogUserNotice("reloading entry")
			return m, reloadEntryCmd()
		case "e":
			m.LogUserNotice("editing entry")
			cmd := m.EditCurrentEntry()
			return m, cmd
		case "up", "k":
			n, err := m.DB.Next(m.Entry)
			if err == db.ErrNoNextEntry {
				return m, nil
			}
			m.handleError("next entry", err)
			m.Entry = n
			return m, nil
		case "down", "j":
			n, err := m.DB.Previous(m.Entry)
			if err == db.ErrNoPrevEntry {
				return m, nil
			}
			m.handleError("previous entry", err)
			m.Entry = n
			return m, nil
		}

	case reloadEntryMsg:
		n, err := m.Reconcile(m.Entry.ID)
		m.handleError("reloaded entry", err)
		m.Entry = n
		return m, repaintCmd()
	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}

func (m *Model) handleError(msg string, err error) {
	if err != nil {
		m.LogUserError(err)
	} else {
		m.LogUserNotice(msg)
	}
}

func (m *Model) findTopMessage() *userMessage {
	if m.messages == nil {
		return nil
	}

	// display any errors as more important to any info

	var infocandidate *userMessage
	for i := len(m.messages) - 1; i >= 0; i-- {
		x := m.messages[i]
		if x.IsError {
			return x
		}
		if x.Time.After(time.Now().Add(-time.Second * 60)) {
			infocandidate = x
			break
		}
	}

	return infocandidate
}

func (m *Model) CurrentEntryPath() string {
	if m.Entry == nil {
		return "no entry"
	}
	return m.DB.StoragePath(m.Entry.ID)
}
