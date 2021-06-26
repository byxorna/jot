package model

import (
	"fmt"
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
			title := e.Title
			completion := EntryTaskCompletion(e)
			titleStyle := lipgloss.NewStyle()
			status := EntryTaskStatus(e)
			if status != "" {
				title = fmt.Sprintf("%s (%s)", title, status)
			}
			if completion >= 1.0 {
				titleStyle = titleStyle.Strikethrough(true)
			}

			if isSameDay(m.Date, e.EntryMetadata.CreationTimestamp) {
				titleStyle = titleStyle.Foreground(lipgloss.Color("#FFF7DB"))
			} else if e.EntryMetadata.CreationTimestamp.Before(m.Date) {
				titleStyle = titleStyle.Foreground(lipgloss.AdaptiveColor{Light: "#969B86", Dark: "#696969"})
			}

			renderedTitle := titleStyle.Render(title)
			if e.ID == m.Entry.ID {
				headerItems = append(headerItems, listActive(renderedTitle))
			} else if completion >= 1.0 {
				headerItems = append(headerItems, listDone(renderedTitle))
			} else if completion > 0.0 {
				headerItems = append(headerItems, listCross(renderedTitle))
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
