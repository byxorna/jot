package calendar

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/runtime"
	"github.com/byxorna/jot/pkg/types"
	v1 "github.com/byxorna/jot/pkg/types/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	// ReconciliationDuration is how often to refresh events from the API
	ReconciliationDuration = time.Minute * 10
	pluginName             = config.PluginTypeCalendar
	// This is sourced from setting up the oauth client somewhere like
	// https://developers.google.com/calendar/caldav/v2/guide?hl=en_US
	// TODO: idk whether its ok to package this into the repo or not!!!!
	//go:embed credentials.json
	credentialsJSON []byte

	// maxEventsInDay is how many events we query from google calendar per day
	maxEventsInDay int64 = 40

	tokenStorageFile = "google_calendar_token.json"
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

// Retrieve a token, saves the token, then returns the generated client.
func getClient(ctx context.Context, oauth2cfg *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile, err := runtime.File(tokenStorageFile)
	if err != nil {
		return nil, fmt.Errorf("unable to determine token storage file %s: %w", tokenStorageFile, err)
	}

	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err := getTokenFromWeb(ctx, oauth2cfg)
		if err != nil {
			return nil, err
		}
		saveToken(tokFile, tok)
	}

	return oauth2cfg.Client(ctx, tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, oauth2cfg *oauth2.Config) (*oauth2.Token, error) {
	authURL := oauth2cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("[plugin:%s] Go to the following link in your browser then type the authorization code: \n%v\n", pluginName, authURL)

	fmt.Printf("[plugin:%s] Please enter the auth code here: ", pluginName)
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := oauth2cfg.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("[plugin:%s] Saving credential file to: %s\n", pluginName, path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}

func New(ctx context.Context) (*Client, error) {
	// If modifying these scopes, delete your previously saved token.json.
	cfg, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	client, err := getClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}

	c := Client{
		Service:     srv,
		calendarIDs: []string{googlePrimaryCalendarID},
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
	c.Lock()
	defer c.Unlock()

	if c.needsReconciliation() {
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