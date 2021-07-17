// Source: https://raw.githubusercontent.com/charmbracelet/glow/master/ui/stashitem.go
package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/ui"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

const (
	verticalLine         = "│"
	fileListingStashIcon = "• "
)

// stashItem wraps any item that is managed by the stash
type stashItem struct {
	// Value we filter against. This exists so that we can maintain positions
	// of filtered items if notes are edited while a filter is active. This
	// field is ephemeral, and should only be referenced during filtering.
	filterValue string

	db.Doc
	db.DocBackend
}

// Generate the value we're doing to filter against.
func (m *stashItem) buildFilterValue() {
	note, err := text.Normalize(m.UnformattedContent())
	if err != nil {
		// log.Printf("error normalizing '%s': %v", m.Content, err)
		m.filterValue = m.UnformattedContent()
	} else {
		m.filterValue = note
	}
}

func stashItemView(commonWidth int, isSelected bool, isFiltering bool, filterText string, visibleItemsCount int, doc db.Doc) string {

	//title / summary / body / links / icon

	var (
		truncateTo   = uint(commonWidth - stashViewHorizontalPadding*2)
		gutter       string
		title        = text.TruncateWithTail(doc.Title(), truncateTo, text.Ellipsis)
		summary      = doc.Summary()
		icon         = doc.Icon()
		matchSnippet = getClosestMatchContextLine(doc.UnformattedContent(), filterText)
	)
	singleFilteredItem := isFiltering && visibleItemsCount == 1

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item
		matchSnippet = ui.DullYellowFg(matchSnippet)
		gutter = ui.DullFuchsiaFg(verticalLine)
		icon = ui.DullFuchsiaFg(icon)
		title = ui.FuchsiaFg(title)
		summary = ui.DullFuchsiaFg(summary)
	} else {
		// Regular (non-selected) items
		gutter = " "
		matchSnippet = ui.BrightGrayFg(matchSnippet)

		if isFiltering && filterText == "" {
			icon = ui.DimGreenFg(icon)
			title = ui.BrightGrayFg(title)
			summary = ui.DimBrightGrayFg(summary)
		} else {
			icon = ui.GreenFg(icon)
			s := termenv.Style{}.Foreground(lib.NewColorPair("#979797", "#847A85").Color())
			title = styleFilteredText(title, filterText, s)
			summary = ui.DimBrightGrayFg(summary)
		}
	}

	lines := []string{
		fmt.Sprintf("%s %s %s", gutter, title, icon),
		fmt.Sprintf("%s %s", gutter, summary),
	}
	if isFiltering && len(filterText) > 2 {
		lines = append(lines, fmt.Sprintf("%s %s", gutter, matchSnippet))
	}
	return strings.Join(lines, "\n")
}

// finds matching context line from the content of haystack and returns it, with
// some buffer on either side to provide interesting context
func getClosestMatchContextLine(haystack, needle string) string {
	if needle == "" {
		return ""
	}
	additonalContext := 15
	maxContextLength := 60
	stacks := []string{}
	for _, line := range strings.Split(haystack, "\n") {
		normalizedHay, err := text.Normalize(line)
		if err != nil {
			return "ERR: " + err.Error()
		}
		stacks = append(stacks, normalizedHay)
	}

	matches := fuzzy.Find(needle, stacks)
	if len(matches) == 0 {
		return ""
	}

	m := matches[0] // only look at the best (first) match

	b := strings.Builder{}
	for i := max(m.MatchedIndexes[0]-additonalContext, 0); i < min(len(m.Str), m.MatchedIndexes[len(m.MatchedIndexes)-1]+additonalContext); i++ {
		b.WriteByte(m.Str[i])
	}

	// trim off any annoying components we may not care about
	res := strings.TrimSpace(b.String())
	return res[0:min(maxContextLength, len(res))]
}

func styleFilteredText(haystack, needles string, defaultStyle termenv.Style) string {
	b := strings.Builder{}

	normalizedHay, err := text.Normalize(haystack)
	if err != nil && debug {
		log.Printf("error normalizing '%s': %v", haystack, err)
	}

	matches := fuzzy.Find(needles, []string{normalizedHay})
	if len(matches) == 0 {
		return defaultStyle.Styled(haystack)
	}

	m := matches[0] // only one match exists
	for i, rune := range []rune(haystack) {
		styled := false
		for _, mi := range m.MatchedIndexes {
			if i == mi {
				b.WriteString(defaultStyle.Underline().Styled(string(rune)))
				styled = true
			}
		}
		if !styled {
			b.WriteString(defaultStyle.Styled(string(rune)))
		}
	}

	return b.String()
}
