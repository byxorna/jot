package v1

import (
	"fmt"
	"strings"
)

var (
	taskIncompleteMarkdown = `- [ ] `
	taskCompleteMarkdown   = `- [x] `
	ellipsis               = "…"
)

type TaskListStatus struct {
	Checked int
	Total   int
}

func (tls *TaskListStatus) String() string {
	return fmt.Sprintf("%s (%d/%d)", tls.PercentString(), tls.Checked, tls.Total)
}

func (tls *TaskListStatus) PercentString() string {
	return fmt.Sprintf("%.f%%", tls.Percent()*100)
}

func (tls *TaskListStatus) Percent() float64 {
	if tls.Total <= 0 {
		return -1.0
	}
	return float64(tls.Checked) / float64(tls.Total)
}

type TaskCompletionStyle string

var (
	TaskStylePercent  TaskCompletionStyle = "percent"
	TaskStyleDiscrete TaskCompletionStyle = "discrete"
)

func TaskList(content string) TaskListStatus {
	nComplete := strings.Count(content, taskCompleteMarkdown)
	nIncomplete := strings.Count(content, taskIncompleteMarkdown)
	return TaskListStatus{
		Checked: nComplete,
		Total:   nComplete + nIncomplete,
	}
}

func (e *Note) NoteTaskStatus(style TaskCompletionStyle) string {
	b := strings.Builder{}
	tls := TaskList(e.Content)
	if tls.Total > 0 {
		nComplete := strings.Count(e.Content, taskCompleteMarkdown)
		nIncomplete := strings.Count(e.Content, taskIncompleteMarkdown)
		pct := tls.Percent()
		if style == TaskStylePercent {
			b.WriteString(fmt.Sprintf("%.f%%", pct*100))
		} else {
			if nComplete+nIncomplete > 0 {
				b.WriteString(fmt.Sprintf("%d/%d", nComplete, nComplete+nIncomplete))
			}
		}
	}
	return b.String()
}
