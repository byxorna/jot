package model

import (
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
			if e.EntryMetadata.CreationTimestamp.Before(m.Entry.CreationTimestamp) {
				headerItems = append(headerItems, listDone(e.Title))
			} else if e.ID == m.Entry.ID {
				headerItems = append(headerItems, listActive(e.Title))
			} else {
				headerItems = append(headerItems, listItem(e.Title))
			}
		}
	}

	return list.Render(lipgloss.JoinVertical(lipgloss.Left, headerItems...)), nil
}
