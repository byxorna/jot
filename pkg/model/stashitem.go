// Source: https://raw.githubusercontent.com/charmbracelet/glow/master/ui/stashitem.go
package model

import (
	"fmt"
	"log"
	"strings"

	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

const (
	verticalLine         = "│"
	fileListingStashIcon = "• "
)

func (md *markdown) colorizedStatus(focused bool) string {
	if md == nil {
		return ""
	}

	pct := EntryTaskCompletion(&md.Entry)
	var colorCompletion = brightGrayFg
	switch {
	case pct >= .95:
		colorCompletion = greenFg
	case pct >= .8:
		colorCompletion = semiDimGreenFg
	case pct >= .4:
		colorCompletion = subtleIndigoFg
	case pct < 0.0:
		colorCompletion = dimBrightGrayFg
	default:
		colorCompletion = faintRedFg
	}

	pctStr := EntryTaskStatus(&md.Entry, TaskStylePercent)
	taskRatio := EntryTaskStatus(&md.Entry, TaskStyleDiscrete)
	rawstatus := fmt.Sprintf("%s (%s)", pctStr, taskRatio)
	if pct < 0.0 {
		rawstatus = "no tasks"
	}
	if !focused {
		return dimBrightGrayFg(rawstatus)
	} else {
		return colorCompletion(rawstatus)
	}
}

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) {

	var (
		truncateTo = uint(m.common.width - stashViewHorizontalPadding*2)
		gutter     string
		title      = md.Title
		date       = md.relativeTime()
		status     = md.colorizedStatus(true)
		icon       = "" //emoji.Scroll.String()
	)

	switch md.docType {
	default:
		title = truncate.StringWithTail(title, truncateTo, ellipsis)
	}

	isSelected := index == m.cursor()
	isFiltering := m.filterState == filtering
	singleFilteredItem := isFiltering && len(m.getVisibleMarkdowns()) == 1

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item

		status = md.colorizedStatus(true) // override the status with a colorized version

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
			//if m.common.latestFileStashed == md.ID &&
			//	m.statusMessage == stashedStatusMessage {
			//	gutter = greenFg(verticalLine)
			//	icon = dimGreenFg(icon)
			//	title = greenFg(title)
			//	date = semiDimGreenFg(date)
			//} else {
			gutter = dullFuchsiaFg(verticalLine)
			icon = dullFuchsiaFg(icon)
			if m.currentSection().key == filterSection &&
				m.filterState == filterApplied || singleFilteredItem {
				s := termenv.Style{}.Foreground(lib.Fuschia.Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
			} else {
				title = fuchsiaFg(title)
			}
			date = dullFuchsiaFg(date)
			//}
		}
	} else {
		// Regular (non-selected) items

		gutter = " "

		//if m.common.latestFileStashed == md.ID &&
		//	m.statusMessage == stashedStatusMessage {
		//	icon = dimGreenFg(icon)
		//	title = greenFg(title)
		//	date = semiDimGreenFg(date)
		//} else
		title = brightGrayFg(title)
		if isFiltering && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			title = dimNormalFg(title)
			date = dimBrightGrayFg(date)
		} else {
			icon = greenFg(icon)
			s := termenv.Style{}.Foreground(lib.NewColorPair("#dddddd", "#1a1a1a").Color())
			title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
			date = dimBrightGrayFg(date)
		}
	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s %s", gutter, status, date)
}

func styleFilteredText(haystack, needles string, defaultStyle, matchedStyle termenv.Style) string {
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
				b.WriteString(matchedStyle.Styled(string(rune)))
				styled = true
			}
		}
		if !styled {
			b.WriteString(defaultStyle.Styled(string(rune)))
		}
	}

	return b.String()
}
