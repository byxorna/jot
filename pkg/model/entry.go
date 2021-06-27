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
	nComplete := strings.Count(e.Content, taskCompleteMarkdown)
	nIncomplete := strings.Count(e.Content, taskIncompleteMarkdown)
	if (nIncomplete + nComplete) <= 0 {
		return 0.0
	}
	return float64(nComplete) / float64(nIncomplete+nComplete)
}

func EntryTaskStatus(e *v1.Entry) string {
	b := strings.Builder{}
	if HasTaskList(e) {
		nComplete := strings.Count(e.Content, taskCompleteMarkdown)
		nIncomplete := strings.Count(e.Content, taskIncompleteMarkdown)
		pct := EntryTaskCompletion(e)
		if pct >= 1.0 {
			b.WriteString(emoji.CheckMarkButton.String())
		} else if nComplete+nIncomplete > 0 {
			b.WriteString(fmt.Sprintf("%d/%d", nComplete, nComplete+nIncomplete))
		}
	}
	return b.String()
}

func (m *Model) RenderEntryMarkdown() (string, error) {
	return mdRenderer.Render(fmt.Sprintf("# %s {#%d}\n\n%s", m.Entry.Title, m.Entry.ID, m.Entry.Content))
}
