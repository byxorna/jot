// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"fmt"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
)

type Mode string

var (
	ViewMode Mode = "view"
	HelpMode Mode = "help"
	EditMode Mode = "edit"
	ListMode Mode = "list"

	UseHighPerformanceRendering = false
)

type Model struct {
	db.DB
	UseAltScreen bool
	content      string

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	Mode     Mode

	messages []*userMessage
	viewport viewport.Model

	// --- glow variables ---
	state    state
	common   *commonModel
	fatalErr error

	// Sub-models
	stash stashModel
	pager pagerModel
}

type userMessage struct {
	// Time is when the message happened
	Time time.Time
	// Message is the terse oneline description of the issue
	Message string
	IsError bool
}

func (m *Model) CurrentEntry() (*v1.Entry, error) {
	md := m.stash.CurrentMarkdown()
	if md == nil {
		return nil, fmt.Errorf("no entry found")
	}
	return &md.Entry, nil
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

	// just blindly return the last message
	return m.messages[len(m.messages)-1]
}

func (m *Model) CurrentEntryPath() string {
	md := m.stash.CurrentMarkdown()
	if md.ID == 0 || md == nil {
		return "no entry"
	}
	return m.DB.StoragePath(md.ID)
}
