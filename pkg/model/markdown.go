// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/markdown.go
package model

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var (
	mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithEmoji(),
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(0))

	hasher                  = fnv.New32a()
	tagColorHashSalt uint32 = 6969420
	// NOTE: changing these dimensions uncovers some awkward indexing issues in the color
	// selection algo for tags. avoid if you can help it
	tagColors = colorGrid(4, 4)
)

// Sort documents with local files first, then by date.
type markdownsByLocalFirst []*stashItem

func (m markdownsByLocalFirst) Len() int      { return len(m) }
func (m markdownsByLocalFirst) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m markdownsByLocalFirst) Less(i, j int) bool {
	// Neither are local files so sort by date descending
	return m[i].Created().After(m[j].Created())
}

func AsStashItem(d db.Doc, backend db.DocBackend) *stashItem {
	i := stashItem{Doc: d, DocBackend: backend}
	return &i
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

func ColoredTags(m db.Doc, joiner string) string {
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
