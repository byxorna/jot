package notion

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/runtime"
	"github.com/byxorna/jot/pkg/types"
	"github.com/dstotijn/go-notion"
	"github.com/go-playground/validator"
)

var (
	reconciliationPeriod = time.Hour * 2
)

type Client struct {
	*notion.Client `validate:"required"`

	//journalDatabase notion.Database `validate:"required"`
	ctx        context.Context `validate:"required"`
	databaseID string          `validate:"required,uuid"`
	status     types.SyncStatus

	// internal fields that are populated by the library
	db               *notion.Database
	pages            []db.Doc
	lastSynchronized time.Time
}

func New(ctx context.Context, settings map[string]string) (*Client, error) {
	databaseID := settings["database"]

	if databaseID == "" {
		return nil, fmt.Errorf("you must provide setting 'database' with the UUID that identifies your database in notion.so. https://www.notion.so/<databaseID>")
	}

	apikeyFile, err := runtime.File("notion_credentials.json")
	if err != nil {
		return nil, fmt.Errorf("unable to determine apikey storage file: %w", err)
	}

	apikey, err := apikeyFromFile(apikeyFile)
	if err != nil {
		apikey, err = apikeyFromUser()
		if err != nil {
			return nil, fmt.Errorf("unable to read apikey from user: %w", err)
		}
		err = saveApikeyToFile(apikeyFile, apikey)
		if err != nil {
			return nil, fmt.Errorf("unable to save apikey: %w", err)
		}
	}

	client := notion.NewClient(apikey)
	c := Client{
		Client:     client,
		ctx:        ctx,
		status:     types.StatusUninitialized,
		databaseID: databaseID,
	}

	validate := validator.New()
	err = validate.Struct(c)
	if err != nil {
		return nil, fmt.Errorf("client failed validation: %w", err)
	}

	return &c, nil
}

func (c *Client) fetchPagesIfNeeded() error {
	if c.db == nil {
		db, err := c.FindDatabaseByID(c.ctx, c.databaseID)
		if err != nil {
			return err
		}
		c.db = &db
	}

	if c.pages == nil || time.Since(c.lastSynchronized) > reconciliationPeriod {
		pages, err := c.refreshPages()
		if err != nil {
			return err
		}
		c.pages = pages
	}

	c.status = types.StatusOK
	return nil
}

func (c *Client) List() ([]db.Doc, error) {
	err := c.fetchPagesIfNeeded()
	if err != nil {
		return nil, err
	}

	return c.pages, nil
}

func (c *Client) refreshPages() ([]db.Doc, error) {
	fmt.Printf("refreshPages()\n")
	c.status = types.StatusSynchronizing
	var cursor string
	var res notion.DatabaseQueryResponse
	var err error
	pages := []db.Doc{}

	sorts := []notion.DatabaseQuerySort{{Timestamp: notion.SortTimeStampCreatedTime, Direction: notion.SortDirDesc}}

	for cursor == "" || res.HasMore {
		fmt.Printf("queryDatabase(%s) %s\n", c.db.ID, cursor)
		res, err = c.QueryDatabase(c.ctx, c.db.ID, &notion.DatabaseQuery{
			Sorts:       sorts,
			StartCursor: cursor,
		})

		if err != nil {
			c.status = types.StatusError
			return nil, err
		}

		for _, page := range res.Results {
			pages = append(pages, &Page{Page: page})
		}

		if res.HasMore && res.NextCursor != nil {
			cursor = *res.NextCursor
		} else {
			// stop the dang merry-go-round
			break
		}
	}
	c.lastSynchronized = time.Now()
	c.status = types.StatusOK
	return pages, nil
}

func (c *Client) Count() int {
	if c.pages == nil {
		return 0
	}
	return len(c.pages)
}

func (c *Client) Get(id types.ID, hardread bool) (db.Doc, error) {
	for _, p := range c.pages {
		if string(id) == p.Identifier().String() {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no page found with id %s", string(id))
}

func (c *Client) DocType() types.DocType {
	return types.NoteDoc
}

func (c *Client) Status() types.SyncStatus {
	return c.status
}

func (c *Client) StoragePath() string {
	return c.databaseID
}

func (c *Client) StoragePathDoc(id types.ID) string {
	return c.databaseID + "?p=" + id.String()
}

func apikeyFromFile(fname string) (string, error) {
	f, err := os.ReadFile(fname)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(f)), nil
}

func saveApikeyToFile(fname string, apikey string) error {
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache apikey to %s: %w", fname, err)

	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", apikey)
	return nil
}

func apikeyFromUser() (string, error) {
	fmt.Printf("\n\nGo to https://www.notion.so/my-integrations and copy your integration's Internal Integration Token here: ")
	var key string
	if _, err := fmt.Scan(&key); err != nil {
		return "", fmt.Errorf("unable to read integration token: %w", err)
	}
	return strings.TrimSpace(key), nil
}
