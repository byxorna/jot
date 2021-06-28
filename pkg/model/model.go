// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"log/syslog"
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
	Err      error
	Mode     Mode

	usermessages []*userMessage
	viewport     viewport.Model
}

type userMessage struct {
	// Time is when the message happened
	Time time.Time
	// ValidFor is how long the message is considered "valid" for
	ValidFor time.Duration
	// Message is the terse oneline description of the issue
	Message string
	// Priority changes how messages are surfaced relative to eachother, and how the UI
	// colors them
	Priority syslog.Priority
}

// AddUserMessage registers an informational message with the app for display
// via UI, logs, whatever
func (m *Model) AddUserError(err error) {
	m.AddUserMessage(err.Error(), syslog.LOG_ERR, Forever)
}

func (m *Model) AddUserMessage(msg string, prio syslog.Priority, duration time.Duration) {
	m.usermessages = append(m.usermessages, &userMessage{
		Time:     time.Now(),
		Message:  msg,
		Priority: prio,
		ValidFor: duration,
	})
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
			m.AddUserMessage("entry view", syslog.LOG_NOTICE, Informational)
			m.Mode = NormalMode
			return m, nil
		case "?":
			m.AddUserMessage("use esc to return", syslog.LOG_NOTICE, Forever)
			m.Mode = HelpMode
			return m, nil
		case "r":
			m.AddUserMessage("reloading entry", syslog.LOG_NOTICE, Informational)
			return m, reloadEntryCmd()
		case "e":
			m.AddUserMessage("editing entry", syslog.LOG_NOTICE, Informational)
			return m, m.EditCurrentEntry()
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
		n, err := m.DB.Get(m.Entry.ID, true)
		m.handleError("reloaded entry", err)
		m.Entry = n
		return m, nil
	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}

func (m *Model) handleError(msg string, err error) {
	if err != nil {
		m.AddUserError(err)
	} else {
		m.AddUserMessage(msg, syslog.LOG_INFO, Informational)
	}
}
