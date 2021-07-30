package text

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	transformer = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
)

//
// finds matching context line from the content of haystack and returns it, with
// some buffer on either side to provide interesting context
// TODO: remove me
func GetClosestMatchContextLine(haystack, needle string) string {
	additonalContext := 15
	maxContextLength := 60
	stacks := []string{}
	for _, line := range strings.Split(haystack, "\n") {
		normalizedHay, err := Normalize(line)
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

func StyleFilteredText(haystack, needles string, defaultStyle termenv.Style) string {
	b := strings.Builder{}

	normalizedHay, _ := Normalize(haystack)

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
func Normalize(in string) (string, error) {
	transformer.Reset()
	out, _, err := transform.String(transformer, in)
	return out, err
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TruncateWithTail(txt string, width uint, ellipsis string) string {
	return truncate.StringWithTail(txt, width, ellipsis)
}
