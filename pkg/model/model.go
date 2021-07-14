// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	config.Config

	UseAltScreen bool
	content      string

	Author   string
	Timeline []time.Time
	Date     time.Time
	Mode     Mode

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

func (m *Model) CurrentEntryPath() string {
	md := m.stash.CurrentMarkdown()
	if md.ID == 0 || md == nil {
		return "no entry"
	}
	return m.DB.StoragePath(md.ID)
}

// Open either the appropriate entry for today, or create a new one
func (m *Model) createDaysEntry() (*Model, tea.Cmd) {
	return m, func() tea.Msg {
		if entries, err := m.DB.ListAll(); err == nil {
			// if the most recent entry isnt the same as our expected filename, create a new entry for today
			expectedFilename := m.Date.Format(fs.StorageFilenameFormat)
			if len(entries) == 0 || len(entries) > 0 && entries[0].CreationTimestamp.Format(fs.StorageFilenameFormat) != expectedFilename {

				// TODO: query for days events and pre-populate them into the content

				var eventContentHeader string
				if calendarPlugin != nil {
					events, err := calendarPlugin.DayEvents(m.Date)
					if err != nil {
						return errMsg{err}
					}

					if len(events) > 0 {
						schedule := strings.Builder{}
						fmt.Fprintf(&schedule, "# Today's Schedule\n")
						for _, e := range events {
							fmt.Fprintf(&schedule, "- [ ] [%s] %v @ %s (%s, %v)\n", e.CalendarList, e.Title, e.Start.Local().Format("15:04"), e.Duration, e.Status)
						}
						fmt.Fprintf(&schedule, "\n")
						eventContentHeader = schedule.String()
					}
				}

				_, err := m.DB.CreateOrUpdateEntry(&v1.Entry{
					EntryMetadata: v1.EntryMetadata{
						Author: m.Author,
						Title:  m.TitleFromTime(m.Date),
						Tags:   m.DefaultTagsForTime(m.Date),
					},
					Content: eventContentHeader + m.Config.EntryTemplate,
				})
				if err != nil {
					return errMsg{fmt.Errorf("unable to create new entry: %w", err)}
				}
				return m.ReloadEntryCollectionCmd()
			} else {
				return m.stash.newStatusMessage(statusMessage{
					status:  normalStatusMessage,
					message: fmt.Sprintf("Entry %s already exists", expectedFilename),
				})
			}
		} else {
			return errMsg{fmt.Errorf("unable to list entries: %w", err)}
		}
	}
}
