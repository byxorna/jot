// Source: https://raw.githubusercontent.com/charmbracelet/glow/master/ui/stashitem.go
package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode"
	"golang.org/x/text/unicode/norm"
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
	note, err := normalize(m.UnformattedContent())
	if err != nil {
		// log.Printf("error normalizing '%s': %v", m.Content, err)
		m.filterValue = m.UnformattedContent()
	} else {
		m.filterValue = note
	}
}

func stashItemView(commonWidth int, isSelected bool, isFiltering bool, filterText string, visibleItemsCount int, si *stashItem) string {
	switch si.Doc.DocType() {
	case types.NoteDoc:
		return stashItemViewNote(commonWidth, isSelected, isFiltering, filterText, visibleItemsCount, si)
	case types.CalendarEntryDoc:
		return stashItemViewCalendar(commonWidth, isSelected, isFiltering, filterText, visibleItemsCount, si)
	default:
		panic(fmt.Sprintf("I have no idea how to render doctype=%s FIXME!!!!", si.Doc.DocType().String()))
	}
}

func stashItemViewNote(commonWidth int, isSelected bool, isFiltering bool, filterText string, visibleItemsCount int, si *stashItem) string {

	var (
		truncateTo   = uint(commonWidth - stashViewHorizontalPadding*2)
		gutter       string
		title        = truncate.StringWithTail(si.Doc.Title(), truncateTo, ellipsis)
		date         = relativeTime(si.Doc.Created())
		icon         = si.Icon()
		tags         = si.ColoredTags(" ")
		matchSnippet = getClosestMatchContextLine(si.UnformattedContent(), filterText)
	)
	singleFilteredItem := isFiltering && visibleItemsCount == 1

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item
		status = si.ColorizedStatus(true) // override the status with a colorized version
		matchSnippet = dullYellowFg(matchSnippet)
		gutter = dullFuchsiaFg(verticalLine)
		icon = dullFuchsiaFg(icon)
		title = fuchsiaFg(title)
		date = dullFuchsiaFg(date)
	} else {
		// Regular (non-selected) items
		gutter = " "
		matchSnippet = brightGrayFg(matchSnippet)

		if isFiltering && filterText == "" {
			icon = dimGreenFg(icon)
			title = brightGrayFg(title)
			date = dimBrightGrayFg(date)
		} else {
			icon = greenFg(icon)
			s := termenv.Style{}.Foreground(lib.NewColorPair("#979797", "#847A85").Color())
			title = styleFilteredText(title, filterText, s)
			date = dimBrightGrayFg(date)
		}
	}

	firstLineLeft := fmt.Sprintf("%s %s %s", gutter, title, icon)
	var firstLineRight string
	if isFiltering {
		firstLineRight = tags
	}
	firstLineSpacer := strings.Repeat(" ", max(0, int(truncateTo)-lipgloss.Width(firstLineLeft)-lipgloss.Width(firstLineRight)))
	secondLineLeft := fmt.Sprintf("%s %s %s %s", gutter, status, date, matchSnippet)
	return fmt.Sprint(firstLineLeft, firstLineSpacer, firstLineRight, "\n", secondLineLeft)
}

// finds matching context line from the content of haystack and returns it, with
// some buffer on either side to provide interesting context
func getClosestMatchContextLine(haystack, needle string) string {
	additonalContext := 15
	maxContextLength := 60
	stacks := []string{}
	for _, line := range strings.Split(haystack, "\n") {
		normalizedHay, err := normalize(line)
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

	normalizedHay, err := normalize(haystack)
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

// Normalize text to aid in the filtering process. In particular, we remove
// diacritics, "ö" becomes "o". Note that Mn is the unicode key for nonspacing
// marks.
func normalize(in string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, in)
	return out, err
}
