package filter

import (
	"fmt"
	"sort"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types"
	"github.com/byxorna/jot/pkg/types/v1"
)

type FilteringBackend struct {
	source         db.DocBackend
	filterSource   func() string
	filterText     string
	cachedFullList []db.Doc
	displayed      []db.Doc
}

func New(filterValue func() string, backend db.DocBackend) (*FilteringBackend, error) {
	b := FilteringBackend{
		source:       backend,
		filterSource: filterValue,
		filterText:   filterValue(),
	}

	err := b.hardPopulate()
	if err != nil {
		return nil, fmt.Errorf("unable to populate filter: %w", err)
	}
	return &b, nil
}

func (b *FilteringBackend) hardPopulate() error {
	docs, err := b.source.List()
	if err != nil {
		return err
	}
	b.cachedFullList = docs
	return nil
}

func (b *FilteringBackend) cachedFilteredList() ([]db.Doc, error) {

	// handle lazily populating from backend until we actually need this data
	if b.cachedFullList == nil {
		err := b.hardPopulate()
		if err != nil {
			return nil, err
		}
	}

	currentFilter := b.filterSource()
	if currentFilter == b.filterText {
		return b.cachedFullList, nil
	}

	if currentFilter == "" {
		b.filterText = currentFilter
		b.displayed = b.cachedFullList
		return b.displayed, nil
	}

	// otherwise, aply filter and sort
	b.filterText = currentFilter

	// TODO try to use the fuzzyfinder again when I can figure out how to make it higher SNR and does not just search for any characters in the filter
	filtered := []db.Doc{}
	for _, d := range b.cachedFullList {
		if d.MatchesFilter(currentFilter) {
			filtered = append(filtered, d)
		}
	}
	// TODO: figure out whether this totally clobbers the ranking that is performed earlier
	// because I would rather the entries stay in order when filtering, instead of sorting by
	// fuzzy finding
	sort.Stable(db.DocsByModified(filtered))
	b.displayed = filtered

	return filtered, nil
}

func (b *FilteringBackend) DocType() types.DocType  { return b.source.DocType() }
func (b *FilteringBackend) List() ([]db.Doc, error) { return b.cachedFilteredList() }
func (b *FilteringBackend) Count() int {
	cfl, err := b.cachedFilteredList()
	if err != nil {
		return -1
	}
	return len(cfl)
}
func (b *FilteringBackend) Status() v1.SyncStatus { return b.source.Status() }
func (b *FilteringBackend) Get(id types.DocIdentifier, hardread bool) (db.Doc, error) {
	return b.source.Get(id, hardread)
}
func (b *FilteringBackend) Reconcile(id types.DocIdentifier) (db.Doc, error) {
	return nil, fmt.Errorf("filter backend is readonly, cannot reconcile %s", id)
}
func (b *FilteringBackend) StoragePath() string { return b.source.StoragePath() }
func (b *FilteringBackend) StoragePathDoc(id types.DocIdentifier) string {
	return b.source.StoragePathDoc(id)
}
