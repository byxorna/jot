package keep

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/plugins"
	"github.com/byxorna/jot/pkg/types"
	keep "google.golang.org/api/keep/v1"
	"google.golang.org/api/option"
)

var (
	// ReconciliationDuration is how often to refresh events from the API
	ReconciliationDuration       = time.Minute * 10
	pluginName                   = plugins.TypeKeep
	pageSize               int64 = 15

	GoogleAuthScopes = []string{keep.KeepScope}
)

type Client struct {
	sync.RWMutex
	*keep.Service

	collection  map[types.ID]*Note
	status      types.SyncStatus
	lastFetched time.Time
}

func New(ctx context.Context, client *http.Client) (*Client, error) { //, client *http.Client) (*Client, error) {
	srv, err := keep.NewService(ctx, option.WithHTTPClient(client), option.WithScopes(GoogleAuthScopes...))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %s client: %w", pluginName, err)
	}

	c := Client{Service: srv}
	return &c, nil
}

func (c *Client) DocType() types.DocType {
	return types.KeepItemDoc
}

func (c *Client) Count() int {
	c.RLock()
	defer c.RUnlock()
	if c.lastFetched.Unix() == 0 {
		return 0
	}
	return len(c.collection)
}

func (c *Client) needsReconciliation() bool {
	return c.lastFetched.Before(time.Now().Add(-ReconciliationDuration)) || c.collection == nil
}

func (c *Client) Get(id types.ID, hardread bool) (db.Doc, error) {
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

	d, ok := c.collection[id]
	if !ok {
		return nil, fmt.Errorf("%s not found", id.String())
	}
	return d, nil
}

func (c *Client) Reconcile(id types.ID) (db.Doc, error) {
	_, err := c.fetchAndPopulateCollection(true)
	if err != nil {
		return nil, err
	}
	return c.Get(id, false)
}

func (c *Client) reconcileNote(id types.ID) (db.Doc, error) {
	c.Lock()
	defer c.Unlock()
	// for now, only synchronize in a readonly manner
	// update the whole collection in a batch

	doc, err := c.Service.Notes.Get(id.String()).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get %s: %w", id, err)
	}
	n := Note{doc}
	c.collection[id] = &n
	return &n, nil
}

func (c *Client) hasDoc(e db.Doc) bool {
	_, ok := c.collection[e.Identifier()]
	return ok
}

func (c *Client) fetchAllNotes() ([]*Note, error) {
	kns, err := c._fetchAllNotes("")
	if err != nil {
		return nil, err
	}
	ns := make([]*Note, len(kns))
	for i, kn := range kns {
		ns[i] = &Note{kn}
	}
	return ns, nil
}

func (c *Client) _fetchAllNotes(pageToken string) ([]*keep.Note, error) {
	aggr := []*keep.Note{}

	// Lists notes using a pagination token.
	res, err := c.Service.Notes.List().PageSize(pageSize).Do()
	if err != nil {
		return nil, err
	}
	aggr = append(aggr, res.Notes...)

	if res.NextPageToken != "" {
		more, err := c._fetchAllNotes(res.NextPageToken)
		if err != nil {
			return nil, err
		}
		aggr = append(aggr, more...)
	}
	return aggr, nil
}

func (c *Client) List() ([]db.Doc, error) {
	return c.fetchAndPopulateCollection(false)
}

func (c *Client) fetchAndPopulateCollection(hardread bool) ([]db.Doc, error) {
	c.Lock()
	defer c.Unlock()

	if c.needsReconciliation() || hardread {
		notes, err := c.fetchAllNotes()
		if err != nil {
			return nil, fmt.Errorf("unable to fetch all keep notes: %w", err)
		}
		c.lastFetched = time.Now()
		c.status = types.StatusOK

		// blow away the prior cache
		newCollection := map[types.ID]*Note{}
		for _, n := range notes {
			newCollection[n.Identifier()] = n
		}
		c.collection = newCollection
	}

	docs := []db.Doc{}
	for _, doc := range c.collection {
		docs = append(docs, db.Doc(doc))
	}
	return docs, nil
}

func (c *Client) StoragePath() string {
	return c.BasePath
}

func (c *Client) StoragePathDoc(id types.ID) string {
	return path.Join(c.BasePath, id.String())
}

func (c *Client) Status() types.SyncStatus {
	if c.needsReconciliation() {
		c.status = types.StatusSynchronizing
	}
	return c.status
}
