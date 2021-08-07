package notion

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	"github.com/dstotijn/go-notion"
)

const (
	ZeroWidthNBSP   = `\uFEFF`
	UTFBOM          = `\xEF\xBB\xBF`
	NamePropertyKey = ZeroWidthNBSP + "Name"
)

type Page struct {
	notion.Page

	fetchBlocksFunc  func() ([]notion.Block, error)
	childBlocksCache []notion.Block
}

func NewPage(p notion.Page, populateBlocks func() ([]notion.Block, error)) Page {
	return Page{
		Page:            p,
		fetchBlocksFunc: populateBlocks,
	}
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
	return p.Page.CreatedTime.Local().Format("2006-01-02 Monday")
}

func (p *Page) Properties() notion.DatabasePageProperties {
	return p.Page.Properties.(notion.DatabasePageProperties)
}

func (p *Page) PropertyKeys() []string {
	keys := []string{}
	for k, _ := range p.Properties() {
		// NOTE(gabe): ive seen odd UTF-8 BOM and zero width NBSP characters make their way
		// into property keys. TODO: we should proactively strip these out?
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p *Page) Summary() (summary string) {
	// get the title, or body snippet
	props := p.Properties()
	v, ok := props["Name"]
	if !ok {
		return ""
	}
	t := ""
	for _, entry := range v.Value().([]notion.RichText) {
		t += entry.Text.Content
	}

	t += " " + strings.Join(p.SelectorTags(), ",")
	return t
}

func (p *Page) SelectorLabels() map[string]string {
	return map[string]string{}
}
func (p *Page) SelectorTags() []string {
	t := []string{}
	val := p.Properties()["Tags"].Value()
	for _, entry := range val.([]notion.SelectOptions) {
		t = append(t, entry.Name)
	}
	return t
}
func (p *Page) AsMarkdown() string {
	if p.childBlocksCache == nil {
		blocks, err := p.fetchBlocksFunc()
		if err != nil {
			fmt.Printf("FUCK: %s\n", err.Error())
			return err.Error()
		}

		p.childBlocksCache = blocks
	}

	return blocks2Markdown(0, p.childBlocksCache...)
}

func richTextAsString(rt ...notion.RichText) string {
	sb := strings.Builder{}
	for _, r := range rt {
		sb.WriteString(r.PlainText)
	}
	return sb.String()
}

func blocks2Markdown(depth int, blocks ...notion.Block) string {
	sb := strings.Builder{}
	padding := strings.Repeat(" ", depth*2)
	for _, b := range blocks {
		switch b.Type {
		case notion.BlockTypeParagraph:
			sb.WriteString(richTextAsString(b.Paragraph.Text...))
		case notion.BlockTypeHeading1:
			sb.WriteString("# " + richTextAsString(b.Heading1.Text...))
		case notion.BlockTypeHeading2:
			sb.WriteString("## " + richTextAsString(b.Heading2.Text...))
		case notion.BlockTypeHeading3:
			sb.WriteString("### " + richTextAsString(b.Heading3.Text...))
		case notion.BlockTypeBulletedListItem:
			sb.WriteString(padding + "- " + richTextAsString(b.BulletedListItem.Text...))
		case notion.BlockTypeNumberedListItem:
			sb.WriteString(padding + "- " + richTextAsString(b.NumberedListItem.Text...))
		case notion.BlockTypeToDo:
			check := " "
			if *b.ToDo.Checked {
				check = "x"
			}
			sb.WriteString(padding + "- [" + check + "] " + richTextAsString(b.ToDo.Text...))
		case notion.BlockTypeToggle:
			sb.WriteString(richTextAsString(b.Toggle.Text...))
		case notion.BlockTypeChildPage:
			sb.WriteString(fmt.Sprintf("[Child Page: %s](%s)", b.ChildPage.Title, b.ChildPage.Title))
		case notion.BlockTypeUnsupported:
			sb.WriteString(fmt.Sprintf("```\nunsupported block type %s\n%v\n```", string(b.Type), b))
		default:
			sb.WriteString(fmt.Sprintf("%v", b))
		}
		sb.WriteString("\n")
		if b.HasChildren {
			sb.WriteString(blocks2Markdown(depth+1, b))
		}
	}
	return sb.String()
}

func (p *Page) ExtraContext() []string {
	return []string{p.Page.URL}
}

func (p *Page) Icon() string {
	// track things based on whether they were changed/created today
	px, py, pz := p.Page.CreatedTime.Date()
	tx, ty, tz := time.Now().Date()
	var istoday, isrecentlyedited bool

	if m := p.Modified(); m != nil && time.Since(*m) < time.Hour*24 {
		isrecentlyedited = true
	}
	if px == tx && py == ty && pz == tz {
		istoday = true
	}

	switch {
	case istoday:
		return text.EmojiSun
	case isrecentlyedited:
		return text.EmojiRecentlyEdited
	default:
		return text.EmojiJournal

	}
	return text.EmojiJournal
}

func (p *Page) Identifier() types.ID {
	return types.ID(p.Page.ID)
}

func (p *Page) Links() map[string]string {
	return map[string]string{
		"self": p.Page.URL,
	}
}

func (p *Page) Validate() error { return nil }

func (p *Page) MatchesFilter(needle string) bool {
	return strings.Contains(p.AsMarkdown(), needle)
}
