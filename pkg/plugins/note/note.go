// Package note is a riff on the fs plugin. It aims to simplify the storage interface
// and more gracefully support structured TODO items
package note

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	"github.com/go-playground/validator"
)

func New(author string, title string, body string, tags []string, labels map[string]string) (*Note, error) {
	creationTimestamp := time.Now().UTC()
	id := IDFromTime(creationTimestamp)
	nn := Note{
		ID:                id,
		CreationTimestamp: creationTimestamp,
		Author:            author,
		TitleX:            title,
		Tags:              tags,
		Labels:            labels,
	}
	if body != "" {
		nn.Content = &Section{
			Text: &TextContent{
				Text: body,
			},
		}
	}

	err := nn.Validate()
	return &nn, err
}

// Note: A single note.
type Note struct {
	Author            string            `json:"author" yaml:"author" validate:"required"`
	Attachments       []*Attachment     `json:"attachments,omitempty" yaml:"attachments,omitempty" validate:""`
	Tags              []string          `json:"tags,omitempty" yaml:"tags,omitempty" validate:""`
	Labels            map[string]string `json:"labels,omitempty" yaml:"labels,omitempty" validate:""`
	Content           *Section          `json:"content,omitempty" yaml:"content" validate:""`
	ID                types.ID          `json:"id" yaml:"id" validate:"required"`
	TitleX            string            `json:"title" yaml:"title" validate:"required"`
	CreationTimestamp time.Time         `json:"created" yaml:"created" validate:"required"`
	TrashedTimestamp  *time.Time        `json:"trashed,omitempty" yaml:"trashed,omitempty" validate:""`
	ModifiedTimestamp *time.Time        `json:"modified,omitempty" yaml:"modified,omitempty" validate:""`
}

// Attachment: An attachment to a note.
type Attachment struct {
	// MimeType: The MIME types (IANA media types) in which the attachment
	// is available.
	MimeType []string `json:"mimeType,omitempty" yaml:"body" validate:""`

	// Name: The resource name;
	Name string `json:"name,omitempty" yaml:"body" validate:""`
}

// Section: The content of the note.
type Section struct {
	// List: Used if this section's content is a list.
	List *ListContent `json:"list,omitempty" yaml:"list,omitempty" validate:""`

	// Text: Used if this section's content is a block of text. The length
	// of the text content must be less than 20,000 characters.
	Text *TextContent `json:"-" yaml:"-" validate:""` // do not load text, we populate it from the body of the loaded markdown
}

// TextContent: The block of text for a single text section or list item.
type TextContent struct {
	// Text: The text of the note. The limits on this vary with the specific
	// field using this type.
	Text string `json:"text,omitempty" yaml:"text,omitempty" validate:""`
}

// ListContent: The list of items for a single list note.
type ListContent struct {
	// ListItems: The items in the list. The number of items must be less than 1,000.
	ListItems []*ListItem `json:"listItems,omitempty" yaml:"listItems" validate:"max=1000"`
}

// ListItem: A single list item in a note's list.
type ListItem struct {
	// Checked: Whether this item has been checked off or not.
	Checked bool `json:"checked,omitempty" yaml:"checked" validate:""`

	// ChildListItems: If set, list of list items nested under this list
	// item. Only one level of nesting is allowed.
	ChildListItems []*ListItem `json:"childListItems,omitempty" yaml:"childListItems,omitempty" validate:""`

	// Text: The text of this item. Length must be less than 1,000
	// characters.
	Text *TextContent `json:"text,omitempty" yaml:"text,omitempty" validate:"required"`
}

func (e *Note) AsMarkdown() (md string) {
	if e.Content == nil {
		return
	}
	return e.Content.AsMarkdown()
}

func (e *Section) AsMarkdown() (md string) {
	if e.List != nil {
		for _, li := range e.List.ListItems {
			md += li.AsMarkdown(0)
		}
	}
	if e.Text != nil {
		if md != "" {
			md += "\n"
		}
		md += e.Text.Text
	}
	return
}

func (e *ListItem) AsMarkdown(indent int) (md string) {
	prefix := "- [ ] "
	if e.Checked {
		prefix = "- [x] "
	}
	md += strings.Repeat(" ", indent) + prefix + strings.ReplaceAll(e.Text.Text, "\n", " ") + "\n"

	if len(e.ChildListItems) > 0 {
		for _, li := range e.ChildListItems {
			md += li.AsMarkdown(indent + 2)
		}
	}
	return
}
func (e *Note) MatchesFilter(needle string) bool {
	return strings.Contains(e.AsMarkdown(), needle)
}
func (e *Note) SelectorLabels() map[string]string { return e.Labels }
func (e *Note) SelectorTags() []string            { return e.Tags }
func (e *Note) Created() time.Time                { return e.CreationTimestamp }
func (e *Note) Modified() *time.Time              { return e.ModifiedTimestamp }
func (e *Note) Trashed() *time.Time               { return e.TrashedTimestamp }
func (e *Note) ExtraContext() []string            { return []string{} }
func (e *Note) Title() string                     { return e.TitleX }
func (e *Note) Context() string                   { return "" }
func (e *Note) Links() map[string]string          { return map[string]string{} }

func (e *Note) listSummary() (total, checked int64, pct float64) {
	if e.Content == nil || e.Content.List == nil || (len(e.Content.List.ListItems) == 0) {
		return 0, 0, 0.0
	}
	total, checked = summarize(e.Content.List.ListItems)
	return total, checked, float64(checked) / float64(total)
}

func summarize(items []*ListItem) (total, checked int64) {
	for _, it := range items {
		total += 1
		if it.Checked {
			checked += 1
		}
		subt, subc := summarize(it.ChildListItems)
		total += subt
		checked += subc
	}
	return total, checked
}

func (e *Note) Summary() (summary string) {
	total, checked, pct := e.listSummary()
	if total == 0 {
		summary = "no tasks"
	} else {

		summary = fmt.Sprintf("%d/%d (%.f%%)", checked, total, pct*100.0)
	}
	relativeAge := text.RelativeTime(e.CreationTimestamp)

	return summary + " " + relativeAge
}

func (e *Note) isCurrentDay() bool {
	return time.Now().Format("2006-01-02") == e.Created().Format("2006-01-02")
}

func (e *Note) Icon() string {
	if e.isCurrentDay() {
		return text.EmojiSun
	}
	return ""
}
func (e *Note) DocType() types.DocType { return types.NoteDoc }
func (e *Note) Validate() error {
	validate := validator.New()
	err := validate.Struct(*e)
	return err
}

func (e *Note) Identifier() types.ID {
	return e.ID
}

type ByCreationTimestampNoteList []*Note

func (p ByCreationTimestampNoteList) Len() int {
	return len(p)
}

func (p ByCreationTimestampNoteList) Less(i, j int) bool {
	return p[i].CreationTimestamp.Before(p[j].CreationTimestamp)
}

func (p ByCreationTimestampNoteList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func IDFromTime(t time.Time) types.ID {
	return types.ID(fmt.Sprintf("%d", t.UTC().Unix()))
}
