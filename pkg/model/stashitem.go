// Source: https://raw.githubusercontent.com/charmbracelet/glow/master/ui/stashitem.go
package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/byxorna/jot/pkg/db"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
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
	note, err := normalize(m.UnformattedContent())
	if err != nil {
		// log.Printf("error normalizing '%s': %v", m.Content, err)
		m.filterValue = m.UnformattedContent()
	} else {
		m.filterValue = note
	}
}

func stashItemView(b *strings.Builder, m stashModel, index int, si *stashItem) {

	var (
		truncateTo   = uint(m.common.width - stashViewHorizontalPadding*2)
		gutter       string
		title        = si.Doc.Title()
		date         = si.relativeTime()
		status       = si.ColorizedStatus(true)
		icon         = si.Icon()
		tags         = ""
		matchSnippet = getClosestMatchContextLine(si.UnformattedContent(), m.filterInput.Value())
	)

	switch si.DocType() {
	default:
		title = truncate.StringWithTail(title, truncateTo, ellipsis)
	}

	isSelected := index == m.cursor()
	isFiltering := m.filterState == filtering
	singleFilteredItem := isFiltering && len(m.getVisibleStashItems()) == 1

	if isFiltering {
		// only show tags in the item entry if filtering is enabled
		tags = si.ColoredTags(" ")
	}

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item

		status = si.ColorizedStatus(true) // override the status with a colorized version
		matchSnippet = dullYellowFg(matchSnippet)

		switch m.selectionState {
		case selectionPromptingDelete:
			gutter = faintRedFg(verticalLine)
			icon = faintRedFg(icon)
			title = redFg(title)
			date = faintRedFg(date)
		case selectionSettingNote:
			gutter = dullYellowFg(verticalLine)
			icon = ""
			title = m.noteInput.View()
			date = dullYellowFg(date)
		default:
			gutter = dullFuchsiaFg(verticalLine)
			icon = dullFuchsiaFg(icon)
			if m.FocusedSection().Identifier() == filterSectionID &&
				m.filterState == filterApplied || singleFilteredItem {
				s := termenv.Style{}.Foreground(lib.Fuschia.Color())
				title = styleFilteredText(title, m.filterInput.Value(), s)
			} else {
				title = fuchsiaFg(title)
			}
			date = dullFuchsiaFg(date)
			//}
		}
	} else {
		// Regular (non-selected) items

		gutter = " "
		matchSnippet = brightGrayFg(matchSnippet)

		if isFiltering && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			title = brightGrayFg(title)
			date = dimBrightGrayFg(date)
		} else {
			icon = greenFg(icon)
			s := termenv.Style{}.Foreground(lib.NewColorPair("#979797", "#847A85").Color())
			title = styleFilteredText(title, m.filterInput.Value(), s)
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
	fmt.Fprint(b,
		firstLineLeft, firstLineSpacer, firstLineRight, "\n",
		secondLineLeft)
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
			return ""
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
	res := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(b.String()), "-"))
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
