// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/markdown.go
package model

import (
	"hash/fnv"
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/byxorna/jot/pkg/types"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/enescakir/emoji"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	hasher                  = fnv.New32a()
	tagColorHashSalt uint32 = 6969420
	// NOTE: changing these dimensions uncovers some awkward indexing issues in the color
	// selection algo for tags. avoid if you can help it
	tagColors = colorGrid(4, 4)
)

// stashItem wraps any item that is managed by the stash
type stashItem struct {
	docType types.DocType

	// Full path of a local markdown file. Only relevant to local documents and
	// those that have been stashed in this session.
	LocalPath string

	// Value we filter against. This exists so that we can maintain positions
	// of filtered items if notes are edited while a filter is active. This
	// field is ephemeral, and should only be referenced during filtering.
	filterValue string

	v1.Note
}

// Generate the value we're doing to filter against.
func (m *stashItem) buildFilterValue() {
	note, err := normalize(m.Content)
	if err != nil {
		if debug {
			log.Printf("error normalizing '%s': %v", m.Content, err)
		}
		m.filterValue = m.Content
	}

	m.filterValue = note
}

// shouldSortAsLocal returns whether or not this markdown should be sorted as though
// it's a local markdown document.
func (m stashItem) shouldSortAsLocal() bool {
	// TODO(gabe): implement this if we have multiple file types
	return m.LocalPath != ""
}

// Sort documents with local files first, then by date.
type markdownsByLocalFirst []*stashItem

func (m markdownsByLocalFirst) Len() int      { return len(m) }
func (m markdownsByLocalFirst) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m markdownsByLocalFirst) Less(i, j int) bool {
	iIsLocal := m[i].shouldSortAsLocal()
	jIsLocal := m[j].shouldSortAsLocal()

	// Local files (and files that used to be local) come first
	if iIsLocal && !jIsLocal {
		return true
	}
	if !iIsLocal && jIsLocal {
		return false
	}

	// Neither are local files so sort by date descending
	if !m[i].Metadata.CreationTimestamp.Equal(m[j].Metadata.CreationTimestamp) {
		return m[i].Metadata.CreationTimestamp.After(m[j].Metadata.CreationTimestamp)
	}

	// If the times also match, sort by unqiue ID.
	// TODO: replace this with simple string sorting via m[i].Identifier()
	ids := v1.ByID{m[i].Metadata.ID, m[j].Metadata.ID}
	sort.Sort(ids)
	return ids[0] == m[i].Metadata.ID
}

func AsStashItem(path string, e v1.Note) stashItem {
	return stashItem{
		docType:   types.NoteDoc,
		LocalPath: path,
		Note:      e,
	}
}

func (m *stashItem) ColoredTags(joiner string) string {
	colorRangeX := len(tagColors)
	colorRangeY := len(tagColors[0])

	var colorizedTags []string
	sortedTags := m.SelectorTags()
	sort.Strings(sortedTags)
	for _, t := range sortedTags {
		// determine what color this tag should be consistently
		hasher.Reset()
		hasher.Write([]byte(t))
		hash := hasher.Sum32() + tagColorHashSalt
		n := colorRangeX * colorRangeY
		idx := hash % uint32(n)
		x := int(idx) / colorRangeX
		y := int(idx) - (x * colorRangeY)

		colorizedTags = append(colorizedTags,
			lipgloss.NewStyle().Foreground(lipgloss.Color(tagColors[x][y])).Render(t))
	}
	return strings.Join(colorizedTags, joiner)
}

func (md *stashItem) ColorizedStatus(focused bool) string {
	if md == nil {
		return ""
	}

	tls := TaskList(md.Content)
	var colorCompletion = brightGrayFg
	pct := tls.Percent()
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

	rawstatus := tls.String()
	if pct < 0.0 {
		rawstatus = "no tasks"
	}
	if !focused {
		return dimBrightGrayFg(rawstatus)
	} else {
		return colorCompletion(rawstatus)
	}
}

func (m *stashItem) IsCurrentDay() bool {
	return time.Now().Format("2006-01-02") == m.Metadata.CreationTimestamp.Format("2006-01-02")
}

func (m *stashItem) Icon() string {
	if m.IsCurrentDay() {
		return emoji.Sun.String()
	}
	return ""
}

func (m *stashItem) relativeTime() string {
	return relativeTime(m.Metadata.CreationTimestamp)
}

// Normalize text to aid in the filtering process. In particular, we remove
// diacritics, "ö" becomes "o". Note that Mn is the unicode key for nonspacing
// marks.
func normalize(in string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, in)
	return out, err
}

// Return the time in a human-readable format relative to the current time.
func relativeTime(then time.Time) string {
	now := time.Now()
	ago := now.Sub(then)
	if ago < time.Minute {
		return "just now"
	} else if ago < humanize.Week {
		return humanize.CustomRelTime(then, now, "ago", "from now", magnitudes)
	}
	return then.Format("02 Jan 2006 15:04 MST")
}

// Magnitudes for relative time.
var magnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1 second %s", DivBy: 1},
	{D: time.Minute, Format: "%d seconds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1 minute %s", DivBy: 1},
	{D: time.Hour, Format: "%d minutes %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1 hour %s", DivBy: 1},
	{D: humanize.Day, Format: "%d hours %s", DivBy: time.Hour},
	{D: 2 * humanize.Day, Format: "1 day %s", DivBy: 1},
	{D: humanize.Week, Format: "%d days %s", DivBy: humanize.Day},
	{D: 2 * humanize.Week, Format: "1 week %s", DivBy: 1},
	{D: humanize.Month, Format: "%d weeks %s", DivBy: humanize.Week},
	{D: 2 * humanize.Month, Format: "1 month %s", DivBy: 1},
	{D: humanize.Year, Format: "%d months %s", DivBy: humanize.Month},
	{D: 18 * humanize.Month, Format: "1 year %s", DivBy: 1},
	{D: 2 * humanize.Year, Format: "2 years %s", DivBy: 1},
	{D: humanize.LongTime, Format: "%d years %s", DivBy: humanize.Year},
	{D: math.MaxInt64, Format: "a long while %s", DivBy: 1},
}
