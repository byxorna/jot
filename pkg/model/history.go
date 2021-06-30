package model

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// EntryHistoryView renders a list of days
func (m *Model) EntryHistoryView() (string, error) {
	md := m.stash.CurrentMarkdown()
	entries, err := m.DB.ListAll()
	if err != nil {
		return "", err
	}

	headerItems := []string{}

	for _, e := range entries {
		if md.ID != 0 {
			title := e.Title
			completion := EntryTaskCompletion(e)
			titleStyle := lipgloss.NewStyle()
			status := EntryTaskStatus(e, TaskStyleDiscrete)
			if status != "" {
				title = fmt.Sprintf("%s (%s)", title, status)
			}
			if completion >= 1.0 {
				titleStyle = titleStyle.Strikethrough(true)
			}

			if e.ID == md.ID {
				titleStyle = titleStyle.Background(subtle)
			}

			if isSameDay(m.Date, e.EntryMetadata.CreationTimestamp) {
				//titleStyle = titleStyle.Foreground(lipgloss.Color("#FFF7DB"))
			} else if e.EntryMetadata.CreationTimestamp.Before(m.Date) {
				titleStyle = titleStyle.Foreground(dim)
			}

			renderedTitle := titleStyle.Render(title)
			if e.ID == md.ID {
				headerItems = append(headerItems, listActive(renderedTitle))
			} else {
				headerItems = append(headerItems, listBullet(renderedTitle))
			}
		}
	}

	return list.Render(lipgloss.JoinVertical(lipgloss.Left, headerItems...)), nil
}

func isSameDay(today time.Time, inQuestion time.Time) bool {
	return today.Format("2006-01-02") == inQuestion.Format("2006-01-02")
}
