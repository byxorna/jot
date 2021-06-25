package model

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// EntryHistoryView renders a list of days
func (m *Model) EntryHistoryView() (string, error) {
	entries, err := m.DB.ListAll()
	if err != nil {
		return "", err
	}

	headerItems := []string{listHeader("Entry History")}
	for _, e := range entries {
		if m.Entry != nil {
			if e.ID == m.Entry.ID {
				headerItems = append(headerItems, listActive(e.Title))
			} else if isSameDay(m.Date, e.EntryMetadata.CreationTimestamp) {
				headerItems = append(headerItems, listCurrent(e.Title))
			} else if e.EntryMetadata.CreationTimestamp.Before(m.Date) {
				headerItems = append(headerItems, listDone(e.Title))
			} else {
				headerItems = append(headerItems, listItem(e.Title))
			}
		}
	}

	return list.Render(lipgloss.JoinVertical(lipgloss.Left, headerItems...)), nil
}

func isSameDay(today time.Time, inQuestion time.Time) bool {
	return today.Format("2006-01-02") == inQuestion.Format("2006-01-02")
}
