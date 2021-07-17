package calendar

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/text"
	"github.com/byxorna/jot/pkg/types"
	"google.golang.org/api/calendar/v3"
)

type Event struct {
	gevent *calendar.Event

	body       string
	created    time.Time
	start      time.Time
	duration   time.Duration
	attendees  []string
	urls       map[string]string
	CalendarID string // what calendar this event is a part of
	Tags       []string
	Labels     map[string]string
}

func (e *Event) Identifier() types.DocIdentifier { return types.DocIdentifier(e.gevent.Id) }
func (e *Event) DocType() types.DocType          { return types.CalendarEntryDoc }
func (e *Event) MatchesFilter(needle string) bool {
	haystack := fmt.Sprintf("%s %s %s", e.Title(), e.Body(), e.Summary())
	return strings.Contains(haystack, needle)
}

func (e *Event) Validate() error                   { return nil }
func (e *Event) SelectorTags() []string            { return e.Tags }
func (e *Event) SelectorLabels() map[string]string { return e.Labels }
func (e *Event) Title() string                     { return e.gevent.Summary }
func (e *Event) Body() string                      { return e.gevent.Description }
func (e *Event) Summary() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("%s, %s", e.start.Local().Format("15:04"), e.duration))
	sb.WriteString(fmt.Sprintf(" (%s)", e.gevent.Status))

	var yes, no, maybe, waiting int
	for _, attendee := range e.gevent.Attendees {
		switch attendee.ResponseStatus {
		case "needsAction":
			waiting += 1
		case "tentative":
			maybe += 1
		case "declined":
			no += 1
		case "accepted":
			yes += 1
		}
	}
	attendeeStatuses := []string{}
	if yes > 0 {
		attendeeStatuses = append(attendeeStatuses, fmt.Sprintf("%d going", yes))
	}
	if no > 0 {
		attendeeStatuses = append(attendeeStatuses, fmt.Sprintf("%d declined", no))
	}
	if maybe > 0 {
		attendeeStatuses = append(attendeeStatuses, fmt.Sprintf("%d maybe", maybe))
	}
	if waiting > 0 {
		attendeeStatuses = append(attendeeStatuses, fmt.Sprintf("%d unread", waiting))
	}

	sb.WriteString(" " + strings.Join(attendeeStatuses, ", "))
	return sb.String()
}

func (e *Event) Icon() string {
	switch e.gevent.Status {
	case "confirmed":
		return text.EmojiCalendar
	case "tentative":
		return text.EmojiThinking
	case "cancelled":
		return text.EmojiCancelled
	default:
		return text.EmojiQuestionmark
	}
}
func (e *Event) Links() map[string]string { return e.urls }
func (e *Event) Created() time.Time       { return e.created }
func (e *Event) Modified() *time.Time     { return nil }
func (e *Event) UnformattedContent() string {
	lines := []string{
		fmt.Sprintf("# **%s** @ %s (%s)\n", e.Title(), e.start.Local().Format("2006-02-01 15:03"), e.duration),
	}
	if e.body != "" {
		lines = append(lines, fmt.Sprintf("> %s\n", e.body))
	}

	lines = append(lines,
		fmt.Sprintf("- Calendar ID: `%s`", e.CalendarID),
		fmt.Sprintf("- Attendees: %s", strings.Join(e.attendees, ", ")))

	keys := make([]string, 0, len(e.urls))
	for k := range e.urls {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("- [%s](%s)", k, e.urls[k]))
	}

	return strings.Join(lines, "\n")
}

func newEvent(calendarID string, item *calendar.Event) (*Event, error) {
	loc, err := time.LoadLocation(item.Start.TimeZone)
	if err != nil {
		return nil, err
	}

	tStart, err := time.ParseInLocation(time.RFC3339, item.Start.DateTime, loc)
	if err != nil {
		return nil, err
	}

	tEnd, err := time.ParseInLocation(time.RFC3339, item.End.DateTime, loc)
	if err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339, item.Created)
	if err != nil {
		return nil, err
	}

	var attendees []string
	for _, a := range item.Attendees {
		attendees = append(attendees, a.Email)
	}
	var urls map[string]string
	{
		if item.HangoutLink != "" {
			urls["Hangout"] = item.HangoutLink
		}
		if item.HtmlLink != "" {
			urls["Event"] = item.HtmlLink
		}
		if item.Gadget != nil && item.Gadget.Link != "" {
			urls["Gadget"] = item.Gadget.Link
		}
	}

	e := Event{
		created:    created,
		start:      tStart,
		duration:   tEnd.Sub(tStart),
		CalendarID: calendarID,
		attendees:  attendees,
		urls:       urls,
		gevent:     item,
	}
	return &e, nil
}

type eventsByCreationDate []*Event

func (c eventsByCreationDate) Len() int      { return len(c) }
func (c eventsByCreationDate) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c eventsByCreationDate) Less(i, j int) bool {
	return c[i].start.Before(c[j].start)
}
