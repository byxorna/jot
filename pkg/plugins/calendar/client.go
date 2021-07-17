package calendar

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types"
	v1 "github.com/byxorna/jot/pkg/types/v1"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	// ReconciliationDuration is how often to refresh events from the API
	ReconciliationDuration = time.Minute * 10
	pluginName             = config.PluginTypeCalendar
	// maxEventsInDay is how many events we query from google calendar per day
	maxEventsInDay int64 = 40

	GoogleAuthScopes = []string{calendar.CalendarEventsReadonlyScope}
)

const (
	googlePrimaryCalendarID = "primary"
)

type Client struct {
	sync.RWMutex
	*calendar.Service

	calendarIDs []string
	status      v1.SyncStatus
	eventList   []*Event
	eventMap    map[types.DocIdentifier]*Event
	lastFetched time.Time
}

func New(ctx context.Context, client *http.Client, settings map[string]string, calendarIDs []string) (*Client, error) {
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}

	if len(calendarIDs) == 0 {
		calendarIDs = []string{googlePrimaryCalendarID}
	}

	c := Client{
		Service:     srv,
		calendarIDs: calendarIDs,
		eventMap:    map[types.DocIdentifier]*Event{},
		eventList:   []*Event{},
	}
	return &c, nil
}

func (c *Client) DayEvents(t time.Time) ([]*Event, error) {
	// search each calendar serially for the events
	aggr := []*Event{}
	for _, calID := range c.calendarIDs {
		events, err := c.dayEvents(t, calID)
		if err != nil {
			return nil, err
		}
		aggr = append(aggr, events...)
	}

	// TODO: filter this or otherwise order based on event time?
	return aggr, nil
}

func (c *Client) dayEvents(t time.Time, calendarID string) ([]*Event, error) {
	tMin := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	// TODO: use the working hours from config instead
	tMax := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 0, 0, time.UTC)

	events, err := c.Service.Events.
		List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(tMin.Format(time.RFC3339)).
		TimeMax(tMax.Format(time.RFC3339)).
		MaxResults(maxEventsInDay).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events from %s calendar: %w", calendarID, err)
	}

	calEntries := make([]*Event, len(events.Items))
	for i, item := range events.Items {
		evt, err := newEvent(calendarID, item)
		if err != nil {
			return nil, err
		}
		calEntries[i] = evt
	}

	return calEntries, nil
}

func (c *Client) DocType() types.DocType {
	return types.CalendarEntryDoc
}

func (c *Client) Count() int {
	c.RLock()
	defer c.RUnlock()
	if c.lastFetched.Unix() == 0 {
		return 0
	}
	return len(c.eventList)
}

func (c *Client) needsReconciliation() bool {
	return c.lastFetched.Before(time.Now().Add(-ReconciliationDuration)) || c.eventList == nil
}

func (c *Client) Get(id types.DocIdentifier, hardread bool) (db.Doc, error) {
	c.Lock()
	defer c.Unlock()

	if hardread {
		// refresh and inject into eventList
		reconciledEvent, err := c.Reconcile(id)
		if err != nil {
			return nil, err
		}
		return reconciledEvent, nil
	}

	for _, e := range c.eventList {
		if e.Identifier() == id {
			return e, nil
		}
	}
	return nil, fmt.Errorf("no event found in cache with id=%s", id)
}

func (c *Client) Reconcile(id types.DocIdentifier) (db.Doc, error) {
	_, err := c.fetchAndPopulateCollection(true)
	if err != nil {
		return nil, err
	}
	return c.Get(id, false)
}

func (c *Client) reconcileSingleEventBroken(id types.DocIdentifier) (db.Doc, error) {
	c.Lock()
	defer c.Unlock()
	// for now, only synchronize in a readonly manner
	// update the eventList
	// if we know about this event, lets figure out its list
	var calendarIDDiscovered string
	for _, e := range c.eventList {
		if e.Identifier() == id {
			// refetch from the same calendar
			// as a performance optimization
			calendarIDDiscovered = e.CalendarID
			break
		}
	}

	if calendarIDDiscovered != "" {
		evt, err := c.Service.Events.Get(calendarIDDiscovered, id.String()).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve event %s from %s calendar: %w", id, calendarIDDiscovered, err)
		}
		e, err := newEvent(calendarIDDiscovered, evt)
		if err != nil {
			return nil, err
		}
		c.addToCollection(e)
		return e, nil
	} else {
		for _, l := range c.calendarIDs {
			evt, err := c.Service.Events.Get(l, id.String()).Do()
			if err == nil {
				e, err := newEvent(calendarIDDiscovered, evt)
				if err != nil {
					return nil, err
				}
				c.addToCollection(e)
				return e, nil
			}
		}
		return nil, fmt.Errorf("no event %s found in any of %v", id, c.calendarIDs)
	}

}

func (c *Client) hasEvent(e *Event) bool {
	_, ok := c.eventMap[e.Identifier()]
	return ok
}

func (c *Client) addToCollection(e *Event) {
	// lock has already been claimed in exported functions
	//
	// insert into list if not present
	if !c.hasEvent(e) {
		c.eventMap[e.Identifier()] = e
		c.eventList = append(c.eventList, e)
	} else {
		for i, x := range c.eventList {
			if x.Identifier() == e.Identifier() {
				c.eventList[i] = e
				c.eventMap[e.Identifier()] = e
			}
		}
	}
	// stable sort list
	sort.Stable(eventsByCreationDate(c.eventList))
}

func (c *Client) List() ([]db.Doc, error) {
	return c.fetchAndPopulateCollection(false)
}

func (c *Client) fetchAndPopulateCollection(hardread bool) ([]db.Doc, error) {
	c.Lock()
	defer c.Unlock()

	if c.needsReconciliation() || hardread {
		events, err := c.DayEvents(time.Now())
		if err != nil {
			return nil, fmt.Errorf("unable to fetch events: %w", err)
		}
		c.lastFetched = time.Now()
		c.status = v1.StatusOK
		c.eventList = events
	}

	docs := make([]db.Doc, len(c.eventList))
	for i, e := range c.eventList {
		docs[i] = db.Doc(e)
	}
	return docs, nil
}

func (c *Client) StoragePath() string {
	return c.BasePath
}

func (c *Client) StoragePathDoc(id types.DocIdentifier) string {
	return path.Join(c.BasePath, id.String())
}

func (c *Client) Status() v1.SyncStatus {
	if c.needsReconciliation() {
		c.status = v1.StatusSynchronizing
	}
	return c.status
}
