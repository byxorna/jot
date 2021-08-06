package notion

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/runtime"
	"github.com/byxorna/jot/pkg/types"
	"github.com/dstotijn/go-notion"
	"github.com/go-playground/validator"
)

type Client struct {
	*notion.Client `validate:"required"`

	//journalDatabase notion.Database `validate:"required"`
	databaseID string `validate:"required,uuid"`
	status     types.SyncStatus
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

func (c *Client) List() ([]db.Doc, error) {
	return nil, fmt.Errorf("fuck")
}

func (c *Client) Count() int {
	return 99
}

func (c *Client) Get(id types.ID, hardread bool) (db.Doc, error) {
	return nil, fmt.Errorf("fixme")
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
