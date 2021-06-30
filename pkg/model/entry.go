package model

import (
	"fmt"
	"strings"

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/glamour"
	"github.com/enescakir/emoji"
)

var (
	taskIncompleteMarkdown = `- [ ] `
	taskCompleteMarkdown   = `- [x] `
	mdRenderer, _          = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithEmoji(),
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(0))
)

func HasTaskList(e *v1.Entry) bool {
	return strings.Contains(e.Content, taskCompleteMarkdown) || strings.Contains(e.Content, taskIncompleteMarkdown)
}

func EntryTaskCompletion(e *v1.Entry) float64 {
	if e == nil {
		return 0.0
	}
	nComplete := strings.Count(e.Content, taskCompleteMarkdown)
	nIncomplete := strings.Count(e.Content, taskIncompleteMarkdown)
	if (nIncomplete + nComplete) <= 0 {
		return 0.0
	}
	return float64(nComplete) / float64(nIncomplete+nComplete)
}

type TaskCompletionStyle string

var (
	TaskStylePercent  TaskCompletionStyle = "percent"
	TaskStyleDiscrete TaskCompletionStyle = "discrete"
)

func EntryTaskStatus(e *v1.Entry, style TaskCompletionStyle) string {
	if e == nil {
		return ""
	}
	b := strings.Builder{}
	if HasTaskList(e) {
		nComplete := strings.Count(e.Content, taskCompleteMarkdown)
		nIncomplete := strings.Count(e.Content, taskIncompleteMarkdown)
		pct := EntryTaskCompletion(e)
		if style == TaskStylePercent {
			b.WriteString(fmt.Sprintf("%3.f%%", pct*100))
		} else {
			if pct >= 1.0 {
				b.WriteString(emoji.CheckMarkButton.String())
			} else if nComplete+nIncomplete > 0 {
				b.WriteString(fmt.Sprintf("%d/%d", nComplete, nComplete+nIncomplete))
			}
		}
	}
	return b.String()
}

//func (m *Model) UpdateContent() error {
//	e, err := m.CurrentEntry()
//	if err != nil {
//		return err
//	}
//	md, err := mdRenderer.Render(e.Content)
//	if err != nil {
//		return err
//	}
//	if m.content != md {
//		m.content = md
//		m.viewport.SetContent(md)
//		m.viewport.YPosition = 0
//	}
//	return nil
//}
