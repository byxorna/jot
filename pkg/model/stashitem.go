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
	md := m.AsMarkdown()
	note, err := text.Normalize(md)
	if err != nil {
		m.filterValue = fmt.Sprintf("!! ERROR: %s\n\n%s", err.Error(), md)
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
		extracontext = doc.ExtraContext()
		icon         = doc.Icon()
		matchSnippet = getClosestMatchContextLine(doc.AsMarkdown(), filterText)
	)

	singleFilteredItem := isFiltering && visibleItemsCount == 1
	hasFocus := isSelected && !isFiltering || singleFilteredItem

	var primaryColor ui.StyleFunc
	var secondaryColor ui.StyleFunc
	var tertiaryColor ui.StyleFunc
	var highlightColor ui.StyleFunc
	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if hasFocus {
		gutter = verticalLine
		primaryColor = ui.FuchsiaFg
		secondaryColor = ui.DimFuchsiaFg
		tertiaryColor = ui.DimGreenFg
		highlightColor = ui.InstaMagenta
	} else {
		// Regular (non-selected) items
		primaryColor = ui.BrightGrayFg
		secondaryColor = ui.DimBrightGrayFg
		tertiaryColor = ui.DimNormalFg
		highlightColor = ui.InstaBlue
		gutter = " "

		if !isFiltering || filterText != "" {
			s := termenv.Style{}.Foreground(lib.NewColorPair("#979797", "#847A85").Color())
			title = styleFilteredText(title, filterText, s)
		}
	}

	lines := []string{
		fmt.Sprintf("%s %s %s", gutter, primaryColor(title), icon),
		fmt.Sprintf("%s %s", gutter, secondaryColor(summary)),
	}
	for _, ctxline := range extracontext {
		lines = append(lines, fmt.Sprintf("%s %s", gutter, tertiaryColor(ctxline)))
	}
	if isFiltering && len(filterText) > 2 {
		lines = append(lines, fmt.Sprintf("%s %s", gutter, highlightColor(matchSnippet)))
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

// TODO: fix me!
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
