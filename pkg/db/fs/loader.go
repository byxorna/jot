package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

var (
	StorageFilenameFormat = "2006-01-02.md"
	StorageGlob           = "*.md"

	ErrUnableToFindMetadataSection = fmt.Errorf("unable to find metadata yaml at header of entry")
)

type Loader struct {
	*sync.Mutex
	Directory string        `yaml"directory" validate:"required,dir"`
	status    v1.SyncStatus `validate:"required"`
	entries   map[v1.ID]*v1.Entry

	mtimeMap map[v1.ID]time.Time
	watcher  *fsnotify.Watcher
}

func New(dir string, createDirIfMissing bool) (*Loader, error) {
	expandedPath, err := homedir.Expand(dir)
	if err != nil {
		return nil, err
	}

	l := Loader{
		Mutex:     &sync.Mutex{},
		Directory: expandedPath,
		status:    v1.StatusUninitialized,
		entries:   map[v1.ID]*v1.Entry{},
		mtimeMap:  map[v1.ID]time.Time{},
	}

	{ // ensure the notes directory is created. TODO should this be part of the fs storage provider
		expandedPath, err := homedir.Expand(l.Directory)
		if err != nil {
			return nil, err
		}

		finfo, err := os.Stat(expandedPath)
		if err != nil || !finfo.IsDir() {
			err := os.Mkdir(expandedPath, 0700)
			if err != nil {
				return nil, fmt.Errorf("error creating %s: %w", l.Directory, err)
			}
		}

		err = l.Validate()
		if err != nil {
			return nil, fmt.Errorf("error validating storage provider: %w", err)
		}

		// Load up all the files we can find at startup
		entryFiles, err := filepath.Glob(path.Join(expandedPath, StorageGlob))
		if err != nil {
			return nil, err
		}
		for _, fn := range entryFiles {
			fmt.Printf("loading %s\n", fn)
			e, err := l.LoadFromFile(fn)
			if err != nil {
				return nil, err
			}
			l.entries[e.ID] = e
		}
	}

	if err := l.startWatcher(); err != nil {
		return nil, fmt.Errorf("unable to create watcher: %w", err)
	}

	l.status = v1.StatusOK

	return &l, nil
}

func (x *Loader) startWatcher() error {
	if x.watcher != nil {
		_ = x.watcher.Close()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(x.Directory)
	if err != nil {
		return fmt.Errorf("unable to watch %s: %w", x.Directory, err)
	}

	x.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				//log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					entries, _ := x.ListAll()
					//fmt.Println("modified file:", event.Name)
					for _, e := range entries {
						if x.StoragePath(e.ID) == event.Name {
							//fmt.Printf("reconciling %d\n", e.ID)
							_, err := x.Reconcile(e.ID)
							if err != nil {
								// TODO: do something better
								fmt.Printf("error reconciling: %v", err)
							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("error:", err)
			}
		}
	}()
	return nil
}

func (x *Loader) Validate() error {
	validate := validator.New()
	err := validate.Struct(*x)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}

// Get loads an entry from disk and caches it in the entry map
func (x *Loader) Get(id v1.ID, forceRead bool) (*v1.Entry, error) {
	if forceRead {
		n, err := x.Reconcile(id)
		if err != nil {
			return nil, fmt.Errorf("unable to reconcile %d: %w", id, err)
		}
		return n, nil
	}

	e, ok := x.entries[id]
	if !ok {
		return nil, db.ErrNoEntryFound
	}
	return e, nil
}

func (x *Loader) CreateOrUpdateEntry(e *v1.Entry) (*v1.Entry, error) {
	x.Lock()
	defer x.Unlock()

	if e.CreationTimestamp.IsZero() {
		e.CreationTimestamp = time.Now()
	}

	if e.ID == 0 {
		e.ID = v1.ID(e.CreationTimestamp.Unix())
	}

	if x.HasEntry(e.ID) {
		t := time.Now()
		e.EntryMetadata.ModifiedTimestamp = &t
	}

	// TODO: union tags and labels with defaults

	if err := x.Write(e); err != nil {
		return nil, fmt.Errorf("unable to store entry %d: %w", e.ID, err)
	}

	x.entries[e.ID] = e

	return e, nil
}

func (x *Loader) LoadFromFile(fileName string) (*v1.Entry, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", fileName, err)
	}

	return x.LoadFromReader(f)
}

func (x *Loader) LoadFromID(id v1.ID) (*v1.Entry, error) {

	return x.LoadFromFile(x.StoragePath(id))
}

func (x *Loader) LoadFromReader(r io.Reader) (*v1.Entry, error) {
	var e v1.Entry

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read: %w", err)
	}

	nChunks := 3
	chunks := strings.SplitN(string(bytes), "---", nChunks)

	if len(chunks) != nChunks {
		return nil, fmt.Errorf("unable to parse metadata section: %w", ErrUnableToFindMetadataSection)
	}

	err = yaml.Unmarshal([]byte(chunks[1]), &e.EntryMetadata)
	if err != nil {
		return nil, fmt.Errorf("unable to deserialize metadata: %w", err)
	}

	e.Content = chunks[2]

	err = e.Validate()
	if err != nil {
		return nil, err
	}

	x.Lock()
	defer x.Unlock()
	x.entries[e.ID] = &e

	return &e, nil
}

func (x *Loader) expandedStoragePath(id v1.ID) string {
	expandedPath, _ := homedir.Expand(x.shortStoragePath(id))
	return expandedPath
}

func (x *Loader) StoragePath(id v1.ID) string {
	return x.expandedStoragePath(id)
}

func (x *Loader) shortStoragePath(id v1.ID) string {
	t := time.Unix(int64(id), int64(0))
	fullPath := path.Join(x.Directory, t.Format(StorageFilenameFormat))
	return fullPath
}

func (x *Loader) Write(e *v1.Entry) error {
	x.status = v1.StatusSynchronizing

	targetpath := x.StoragePath(e.ID)
	finfo, err := os.Stat(targetpath)
	if err == nil && finfo.IsDir() {
		err := os.RemoveAll(targetpath)
		if err != nil {
			x.status = v1.StatusError
			return err
		}
	}

	f, err := os.OpenFile(targetpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		x.status = v1.StatusError
		return err
	}
	defer f.Close()

	metadata, err := yaml.Marshal(e.EntryMetadata)
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to marshal entry metadata for %d: %w", e.ID, err)
	}

	_, err = f.WriteString(fmt.Sprintf("---\n%s\n---\n", metadata))
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to write entry metadata for %d: %w", e.ID, err)
	}

	_, err = f.WriteString(e.Content + "\n")
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to write entry %d: %w", e.ID, err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("unable to sync entry %d: %w", e.ID, err)
	}

	x.status = v1.StatusOK
	return nil
}

// ListAll returns entries in newest to oldest order
func (x *Loader) ListAll() ([]*v1.Entry, error) {
	x.Lock()
	defer x.Unlock()

	sorted := []*v1.Entry{}
	for _, e := range x.entries {
		sorted = append(sorted, e)
	}
	sort.Sort(sort.Reverse(v1.ByCreationTimestampEntryList(sorted)))
	return sorted, nil
}

func (x *Loader) idx(list []*v1.Entry, e *v1.Entry) (int, error) {

	for i, o := range list {
		if e == o {
			return i, nil
		}
	}
	return 0, db.ErrNoEntryFound
}

func (x *Loader) Next(e *v1.Entry) (*v1.Entry, error) {
	// TODO: this is super slow, i know. ill make it faster after PoC
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	i, err := x.idx(elements, e)
	if err != nil {
		return nil, err
	}

	nextIdx := i - 1
	if nextIdx < 0 || nextIdx >= len(elements) || elements[nextIdx] == nil {
		return nil, db.ErrNoNextEntry
	}
	return elements[nextIdx], nil
}

func (x *Loader) Previous(e *v1.Entry) (*v1.Entry, error) {
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	i, err := x.idx(elements, e)
	if err != nil {
		return nil, err
	}

	prevIdx := i + 1
	if prevIdx < 0 || prevIdx >= len(elements) || elements[prevIdx] == nil {
		return nil, db.ErrNoPrevEntry
	}
	return elements[prevIdx], nil
}

func (x *Loader) Count() int {
	x.Lock()
	defer x.Unlock()
	return len(x.entries)
}

func (x *Loader) HasEntry(id v1.ID) bool {
	_, ok := x.entries[id]
	return ok
}

func (x *Loader) Status() v1.SyncStatus {
	return x.status
}

func (x *Loader) Reconcile(id v1.ID) (*v1.Entry, error) {
	// stat the file on disk, compare to last known mtime. if more recent
	// reload
	if !x.HasEntry(id) || x.ShouldReloadFromDisk(id) {
		e, err := x.LoadFromID(id)
		if err != nil {
			return nil, err
		}
		x.entries[id] = e
	}

	return x.entries[id], nil
}

func (x *Loader) ShouldReloadFromDisk(id v1.ID) bool {
	finfo, err := os.Stat(x.StoragePath(id))
	if err != nil {
		return true
	}

	if x.mtimeMap[id].Before(finfo.ModTime()) {
		return true
	}
	x.mtimeMap[id] = finfo.ModTime()

	return false
}
