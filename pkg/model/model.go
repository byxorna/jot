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
)

type Model struct {
	db.DB
	viewportReady bool
	content       string

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	EntryID  v1.ID
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

func (m *Model) CurrentEntry() (*v1.Entry, error) {
	return m.DB.Get(m.EntryID, false)
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
		if !m.viewportReady {
			// Since this program is using the full size of the viewport we need
			// to wait until we've received the window dimensions before we
			// can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.Model{Width: msg.Width, Height: msg.Height}
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = false
			m.viewport.SetContent(m.content)
			m.viewportReady = true
		}
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
			e, err := m.DB.Next(m.EntryID)
			if err == db.ErrNoNextEntry {
				return m, nil
			}
			m.handleError("next entry", err)
			m.EntryID = e.ID
			return m, updateViewCmd()
		case "down", "j":
			e, err := m.DB.Previous(m.EntryID)
			if err == db.ErrNoPrevEntry {
				return m, nil
			}
			m.handleError("previous entry", err)
			m.EntryID = e.ID
			return m, updateViewCmd()
		}

	case updateViewMsg:
		err := m.UpdateContent()
		if err != nil {
			m.LogUserError(err)
		}
		return m, nil
	case reloadEntryMsg:
		_, err := m.Reconcile(m.EntryID)
		if err != nil {
			m.LogUserError(err)
		}
		return m, updateViewCmd()
	}

	// Because we're using the viewport's default update function (with pager-
	// style navigation) it's important that the viewport's update function:
	//
	// * Receives messages from the Bubble Tea runtime
	// * Returns commands to the Bubble Tea runtime
	//
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
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
	if m.EntryID == 0 {
		return "no entry"
	}
	return m.DB.StoragePath(m.EntryID)
}
