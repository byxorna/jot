package text

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/enescakir/emoji"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	Ellipsis = "…"
)

var (
	EmojiSun            = emoji.Sun.String()
	EmojiCalendar       = emoji.Calendar.String()
	EmojiQuestionmark   = emoji.QuestionMark.String()
	EmojiThinking       = emoji.ThinkingFace.String()
	EmojiCancelled      = emoji.CrossMark.String()
	EmojiNote           = emoji.Notebook.String()
	EmojiComplete       = emoji.CheckBoxWithCheck.String()
	EmojiIncomplete     = emoji.ConstructionWorker.String()
	EmojiRecentlyEdited = emoji.SpiralNotepad.String()
	EmojiJournal        = emoji.NotebookWithDecorativeCover.String()
)

var (
	hasher                  = fnv.New32a()
	tagColorHashSalt uint32 = 6969420
	// NOTE: changing these dimensions uncovers some awkward indexing issues in the color
	// selection algo for tags. avoid if you can help it
	tagColors = colorGrid(4, 4)
)

// Return the time in a human-readable format relative to the current time.
func RelativeTime(then time.Time) string {
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

func ColoredTags(tags []string, joiner string) string {
	colorRangeX := len(tagColors)
	colorRangeY := len(tagColors[0])

	var colorizedTags []string
	sortedTags := tags
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

func colorGrid(xSteps, ySteps int) [][]string {
	x0y0, _ := colorful.Hex("#F25D94")
	x1y0, _ := colorful.Hex("#EDFF82")
	x0y1, _ := colorful.Hex("#643AFF")
	x1y1, _ := colorful.Hex("#14F9D5")

	x0 := make([]colorful.Color, ySteps)
	for i := range x0 {
		x0[i] = x0y0.BlendLuv(x0y1, float64(i)/float64(ySteps))
	}

	x1 := make([]colorful.Color, ySteps)
	for i := range x1 {
		x1[i] = x1y0.BlendLuv(x1y1, float64(i)/float64(ySteps))
	}

	grid := make([][]string, ySteps)
	for x := 0; x < ySteps; x++ {
		y0 := x0[x]
		grid[x] = make([]string, xSteps)
		for y := 0; y < xSteps; y++ {
			grid[x][y] = y0.BlendLuv(x1[x], float64(y)/float64(xSteps)).Hex()
		}
	}

	return grid
}
