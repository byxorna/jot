package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	"github.com/enescakir/emoji"
	"github.com/go-playground/validator"
)

type Note struct {
	Metadata NoteMetadata `yaml:"metadata" validate:"required"`
	Content  string       `yaml:"content" validate:""`
}

type NoteMetadata struct {
	ID                ID                `yaml:"id" validate:"required"`
	Author            string            `yaml:"author" validate:"required"`
	Title             string            `yaml:"title,omitempty" validate:""`
	CreationTimestamp time.Time         `yaml:"created" validate:"required"`
	ModifiedTimestamp *time.Time        `yaml:"modified,omitempty" validate:""`
	Tags              []string          `yaml:"tags,omitempty,flow" validate:""`
	Labels            map[string]string `yaml:"labels,omitempty,flow" validate:""`
}

type ByCreationTimestampNoteList []*Note

func (p ByCreationTimestampNoteList) Len() int {
	return len(p)
}

func (p ByCreationTimestampNoteList) Less(i, j int) bool {
	return p[i].Metadata.CreationTimestamp.Before(p[j].Metadata.CreationTimestamp)
}

func (p ByCreationTimestampNoteList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (e *Note) Validate() error {
	validate := validator.New()
	err := validate.Struct(*e)
	return err
}

func (e *Note) Identifier() types.DocIdentifier {
	return types.DocIdentifier(fmt.Sprintf("%d", e.Metadata.ID))
}

func (e *Note) MatchesFilter(needle string) bool  { return strings.Contains(e.Content, needle) }
func (e *Note) DocType() types.DocType            { return types.NoteDoc }
func (e *Note) SelectorLabels() map[string]string { return e.Metadata.Labels }
func (e *Note) SelectorTags() []string            { return e.Metadata.Tags }
func (e *Note) UnformattedContent() string        { return e.Content }
func (e *Note) Title() string                     { return e.Metadata.Title }
func (e *Note) Created() time.Time                { return e.Metadata.CreationTimestamp }
func (e *Note) Modified() *time.Time              { return e.Metadata.ModifiedTimestamp }
func (e *Note) Body() string                      { return e.Content }
func (e *Note) Links() map[string]string          { return map[string]string{} }

func (e *Note) Summary() string {
	var rawstatus string
	tls := TaskList(e.UnformattedContent())
	pct := tls.Percent()

	rawstatus = tls.String()
	if pct < 0.0 {
		rawstatus = "no tasks"
	}

	relativeAge := text.RelativeTime(e.Metadata.CreationTimestamp)

	return rawstatus + " " + relativeAge
}

func (e *Note) IsCurrentDay() bool {
	return time.Now().Format("2006-01-02") == e.Created().Format("2006-01-02")
}

func (e *Note) Icon() string {
	if e.IsCurrentDay() {
		return emoji.Sun.String()
	}
	return ""
}
