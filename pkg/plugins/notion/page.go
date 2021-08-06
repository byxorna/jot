package notion

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	"github.com/dstotijn/go-notion"
)

type Page struct {
	notion.Page
}

func (p *Page) DocType() types.DocType {
	return types.NoteDoc
}

func (p *Page) Created() time.Time {
	return p.Page.CreatedTime
}

func (p *Page) Modified() *time.Time {
	return &p.Page.LastEditedTime
}

func (p *Page) Trashed() *time.Time {
	if !p.Page.Archived {
		return nil
	}
	return &p.Page.LastEditedTime
}

func (p *Page) Title() string {
	props := p.Page.Properties.(notion.DatabasePageProperties)
	fields := (props["title"].Value()).([]notion.RichText)
	var t string
	for _, f := range fields {
		t += f.PlainText
	}
	return t
}
func (p *Page) Summary() (summary string)         { return "summary here" }
func (p *Page) SelectorLabels() map[string]string { return map[string]string{} }
func (p *Page) SelectorTags() []string {
	//props := p.Page.Properties.(notion.DatabasePageProperties)
	// TODO
	return []string{}
}
func (p *Page) AsMarkdown() string {
	props := p.Page.Properties.(notion.DatabasePageProperties)
	return fmt.Sprintf("%+v", props)
}

func (p *Page) ExtraContext() []string {
	return []string{}
}

func (p *Page) Icon() string {
	px, py, pz := p.Page.CreatedTime.Date()
	tx, ty, tz := time.Now().Date()
	if px == tx && py == ty && pz == tz {
		return text.EmojiSun
	}
	return ""
}

func (p *Page) Identifier() types.ID {
	return types.ID(p.Page.ID)
}
func (p *Page) Links() map[string]string { return map[string]string{} }
func (p *Page) Validate() error          { return nil }
func (p *Page) MatchesFilter(needle string) bool {
	return strings.Contains(p.AsMarkdown(), needle)
}
