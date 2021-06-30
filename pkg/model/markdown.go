// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/markdown.go
package model

import (
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// markdown wraps v1.Entry
type markdown struct {
	docType DocType

	// Full path of a local markdown file. Only relevant to local documents and
	// those that have been stashed in this session.
	localPath string

	// Value we filter against. This exists so that we can maintain positions
	// of filtered items if notes are edited while a filter is active. This
	// field is ephemeral, and should only be referenced during filtering.
	filterValue string

	v1.Entry
}

// Generate the value we're doing to filter against.
func (m *markdown) buildFilterValue() {
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
func (m markdown) shouldSortAsLocal() bool {
	// TODO(gabe): implement this if we have multiple file types
	return m.localPath != ""
}

// Sort documents with local files first, then by date.
type markdownsByLocalFirst []*markdown

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

	// If both are local files, sort by filename. Note that we should never
	// hit equality here since two files can't have the same path.
	if iIsLocal && jIsLocal {
		return strings.Compare(m[i].localPath, m[j].localPath) == -1
	}

	// Neither are local files so sort by date descending
	if !m[i].CreationTimestamp.Equal(m[j].CreationTimestamp) {
		return m[i].CreationTimestamp.After(m[j].CreationTimestamp)
	}

	// If the times also match, sort by unqiue ID.
	ids := v1.ByID{m[i].ID, m[j].ID}
	sort.Sort(ids)
	return ids[0] == m[i].ID
}

func AsMarkdown(path string, e v1.Entry) *markdown {
	return &markdown{
		docType:   LocalDoc,
		localPath: path,
		Entry:     e,
	}
}

//func wrapMarkdowns(md []*v1.Entry) (m []*markdown) {
//	for _, v := range md {
//		m = append(m, &markdown{
//			docType:  LocalDoc,
//      localPath: v.Sto
//			Entry: v,
//		})
//	}
//	return m
//}

func (m *markdown) relativeTime() string {
	return relativeTime(m.CreationTimestamp)
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
