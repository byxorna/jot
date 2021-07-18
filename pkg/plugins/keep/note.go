package keep

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	keep "google.golang.org/api/keep/v1"
)

type Note struct {
	*keep.Note
}

func (n *Note) Icon() string {
	return text.EmojiNote
}
func (n *Note) DocType() types.DocType {
	return types.KeepItemDoc
}

func (n *Note) Identifier() types.DocIdentifier {
	return types.DocIdentifier(n.Name)
}

func (n *Note) SelectorLabels() map[string]string {
	return map[string]string{}
}

func (n *Note) Validate() error {
	return nil
}

func (n *Note) SelectorTags() []string {
	return []string{}
}

func (n *Note) Created() time.Time {
	t, err := time.Parse(time.RFC3339, n.UpdateTime)
	if err != nil {
		panic(err)
	}
	return t
}

func (n *Note) Modified() *time.Time {
	t, err := time.Parse(time.RFC3339, n.UpdateTime)
	if err != nil {
		panic(err)
	}
	return &t
}

func (n *Note) UnformattedContent() string {
	return n.Body()
}

func (n *Note) Title() string {
	return n.Note.Title
}

func (n *Note) MatchesFilter(needle string) bool {
	return strings.Contains(n.UnformattedContent(), needle)
}

func (n *Note) Links() map[string]string {
	links := map[string]string{}
	for _, l := range n.Note.Attachments {
		links[l.Name] = l.Name
	}
	return links
}

func (n *Note) Body() string {
	if len(n.Note.Body.List.ListItems) > 0 {
		return renderList(n.Note.Body.List.ListItems)
	} else {
		return n.Note.Body.Text.Text
	}
}

func listSummary(listItems []*keep.ListItem) string {
	c, u := _listSummary(listItems)
	return fmt.Sprintf("%d/%d (%03.f%%)", c, c+u, (c*1.0)/((c+u)*1.)*100.)
}

func _listSummary(items []*keep.ListItem) (checked, unchecked int) {
	for _, i := range items {
		if i.Checked {
			checked += 1
		} else {
			unchecked += 1
		}
		subc, subu := _listSummary(i.ChildListItems)
		checked += subc
		unchecked += subu
	}
	return
}

func renderList(listItems []*keep.ListItem) string {
	return strings.Join(_renderList(listItems, 0), "\n")
}

func _renderList(listItems []*keep.ListItem, indentation int) []string {
	res := []string{}
	for _, i := range listItems {

		var marked = " "
		if i.Checked {
			marked = "x"
		}
		res = append(res, fmt.Sprintf("%s- [%s] %s", strings.Repeat(" ", indentation), marked, i.Text.Text))
		children := _renderList(i.ChildListItems, indentation+2)
		for _, subi := range children {
			res = append(res, subi)
		}
	}
	return res
}

func (n *Note) ExtraContext() []string {
	sb := strings.Builder{}
	k := n.Note
	sb.WriteString("created " + k.CreateTime)
	sb.WriteString(", ")
	sb.WriteString("modified " + k.UpdateTime)
	return []string{sb.String()}
}

func (n *Note) Summary() string {
	sb := strings.Builder{}
	k := n.Note
	if len(k.Attachments) > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprintf("%d attachments", len(k.Attachments)))
	}
	if len(k.Body.List.ListItems) > 0 {
		sb.WriteString(", " + listSummary(k.Body.List.ListItems))
	}
	return sb.String()
}
