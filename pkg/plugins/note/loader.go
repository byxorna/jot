package note

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

var (
	StorageFilenameFormat = "2006-01-02.md"
	StorageGlob           = "*.md"

	ErrUnableToFindMetadataSection = fmt.Errorf("unable to find metadata yaml at header of note")

	docTypes = types.NewDocTypeSet(types.NoteDoc)
)

type Store struct {
	*sync.Mutex

	Directory string `yaml:"directory" validate:"required,dir"`

	status   types.SyncStatus
	entries  map[types.ID]*Note
	mtimeMap map[types.ID]time.Time
	watcher  *fsnotify.Watcher
}

func NewStore(dir string, createDirIfMissing bool) (*Store, error) {
	expandedPath, err := homedir.Expand(dir)
	if err != nil {
		return nil, err
	}

	s := Store{
		Mutex:     &sync.Mutex{},
		Directory: expandedPath,
		status:    types.StatusUninitialized,
		entries:   map[types.ID]*Note{},
		mtimeMap:  map[types.ID]time.Time{},
	}

	{ // ensure the notes directory is created. TODO should this be part of the fs storage provider
		expandedPath, err := homedir.Expand(s.Directory)
		if err != nil {
			return nil, err
		}

		finfo, err := os.Stat(expandedPath)
		if err != nil || !finfo.IsDir() {
			err := os.Mkdir(expandedPath, 0700)
			if err != nil {
				return nil, fmt.Errorf("error creating %s: %w", s.Directory, err)
			}
		}

		err = s.Validate()
		if err != nil {
			return nil, fmt.Errorf("error validating storage provider: %w", err)
		}

		// Load up all the files we can find at startup
		noteFiles, err := filepath.Glob(path.Join(expandedPath, StorageGlob))
		if err != nil {
			return nil, err
		}
		for _, fn := range noteFiles {
			//fmt.Fprintf(os.Stderr, "loading %s\n", fn)
			e, err := s.LoadFromFile(fn)
			if err != nil {
				return nil, fmt.Errorf("error loading %s: %w", fn, err)
			}
			s.entries[e.ID] = e
		}
	}

	//disabled 2021.07.30
	//if err := s.startWatcher(); err != nil {
	//		return nil, fmt.Errorf("unable to watch %s: %w", s.Directory, err)
	//	}

	s.status = types.StatusOK

	return &s, nil
}

func (x *Store) startWatcher() error {
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

				fmt.Fprintf(os.Stderr, "event: %v\n", event)
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Fprintln(os.Stderr, "modified file:", event.Name)
					entries, _ := x.ListAll()
					for _, e := range entries {
						expectedFileName := createdTimeToFileName(e.CreationTimestamp)
						if expectedFileName == path.Base(event.Name) {
							fmt.Fprintf(os.Stderr, "reconciling %s\n", event.Name)
							_, err := x.Reconcile(e.ID)
							if err != nil {
								// TODO: do something better
								fmt.Fprintf(os.Stderr, "error reconciling %s: %v\n", e.ID, err)
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

func (x *Store) Validate() error {
	validate := validator.New()
	err := validate.Struct(*x)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}

func parseID(id types.ID) (*time.Time, error) {
	id64, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse note ID %v: %w", id, err)
	}

	t := time.Unix(id64, 0)
	return &t, err
}

func (x *Store) Get(id types.ID, hardread bool) (db.Doc, error) {
	if hardread {
		n, err := x.Reconcile(id)
		if err != nil {
			return nil, fmt.Errorf("unable to reconcile %d: %w", id, err)
		}
		return n, nil
	}

	e, ok := x.entries[id]
	if !ok {
		return nil, db.ErrNoNoteFound
	}
	return e, nil
}

func (x *Store) CreateOrUpdateNote(e *Note) (*Note, error) {
	x.Lock()
	defer x.Unlock()

	if e.CreationTimestamp.IsZero() {
		e.CreationTimestamp = time.Now()
	}

	if e.ID == "" {
		e.ID = types.ID(fmt.Sprintf("%d", e.CreationTimestamp.Unix()))
	}

	// TODO: union tags and labels with defaults

	if err := x.Write(e); err != nil {
		return nil, fmt.Errorf("unable to store note %d: %w", e.ID, err)
	}

	x.entries[e.ID] = e

	return e, nil
}

func (x *Store) LoadFromFile(fileName string) (*Note, error) {
	if !strings.HasPrefix(fileName, x.StoragePath()) {
		return nil, fmt.Errorf("file %s does not begin with %s", fileName, x.StoragePath())
	}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", fileName, err)
	}
	defer f.Close()

	return x.LoadFromReader(f)
}

func (x *Store) LoadFromReader(r io.Reader) (*Note, error) {
	var e Note

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read: %w", err)
	}

	nChunks := 3
	chunks := strings.SplitN(string(bytes), "---", nChunks)

	if len(chunks) != nChunks {
		return nil, fmt.Errorf("unable to parse metadata section: %w", ErrUnableToFindMetadataSection)
	}

	err = yaml.Unmarshal([]byte(chunks[1]), &e)
	if err != nil {
		return nil, fmt.Errorf("unable to deserialize metadata: %w", err)
	}

	if chunks[2] != "" {
		if e.Content == nil {
			e.Content = &Section{}
		}
		e.Content.Text = &TextContent{
			Text: chunks[2],
		}
	}

	err = e.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	x.Lock()
	defer x.Unlock()
	x.entries[e.ID] = &e

	return &e, nil
}

func (x *Store) StoragePath() string {
	expandedPath, _ := homedir.Expand(x.Directory)
	return expandedPath
}

func (x *Store) StoragePathDoc(id types.ID) string {
	t, err := parseID(id)
	if err != nil {
		panic(err)
	}
	fullPath := path.Join(x.Directory, createdTimeToFileName(*t))
	return fullPath
}

func createdTimeToFileName(t time.Time) string {
	return t.UTC().Format(StorageFilenameFormat)
}

func (x *Store) Write(e *Note) error {
	x.status = types.StatusSynchronizing

	targetpath := x.StoragePathDoc(e.Identifier())
	finfo, err := os.Stat(targetpath)
	if err == nil && finfo.IsDir() {
		err := os.RemoveAll(targetpath)
		if err != nil {
			x.status = types.StatusError
			return err
		}
	}

	f, err := os.OpenFile(targetpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		x.status = types.StatusError
		return err
	}
	defer f.Close()

	metadata, err := yaml.Marshal(e)
	if err != nil {
		x.status = types.StatusError
		return fmt.Errorf("unable to marshal note metadata for %s: %w", e.ID, err)
	}

	_, err = f.WriteString(fmt.Sprintf("---\n%s\n---\n", strings.TrimSpace(string(metadata))))
	if err != nil {
		x.status = types.StatusError
		return fmt.Errorf("unable to write note metadata for %s: %w", e.ID, err)
	}

	if e.Content.Text != nil {
		_, err = f.WriteString(e.Content.Text.Text)
		if err != nil {
			x.status = types.StatusError
			return fmt.Errorf("unable to write note %s body: %w", e.ID, err)
		}
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("unable to sync note %s: %w", e.ID, err)
	}

	x.status = types.StatusOK
	return nil
}

// ListAll returns entries in newest to oldest order
func (x *Store) ListAll() ([]*Note, error) {
	x.Lock()
	defer x.Unlock()

	sorted := []*Note{}
	for _, e := range x.entries {
		sorted = append(sorted, e)
	}
	sort.Sort(sort.Reverse(ByCreationTimestampNoteList(sorted)))
	return sorted, nil
}

func (x *Store) idx(list []*Note, id types.ID) (int, error) {

	for i, o := range list {
		if id == o.ID {
			return i, nil
		}
	}
	return 0, db.ErrNoNoteFound
}

func (x *Store) Next(id types.ID) (*Note, error) {
	// TODO: this is super slow, i know. ill make it faster after PoC
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	i, err := x.idx(elements, id)
	if err != nil {
		return nil, err
	}

	nextIdx := i - 1
	if nextIdx < 0 || nextIdx >= len(elements) || elements[nextIdx] == nil {
		return nil, db.ErrNoNextNote
	}
	return elements[nextIdx], nil
}

func (x *Store) Previous(id types.ID) (*Note, error) {
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	i, err := x.idx(elements, id)
	if err != nil {
		return nil, err
	}

	prevIdx := i + 1
	if prevIdx < 0 || prevIdx >= len(elements) || elements[prevIdx] == nil {
		return nil, db.ErrNoPrevNote
	}
	return elements[prevIdx], nil
}

func (x *Store) Count() int {
	x.Lock()
	defer x.Unlock()
	return len(x.entries)
}

func (x *Store) HasNote(id types.ID) bool {
	_, ok := x.entries[id]
	return ok
}

func (x *Store) Status() types.SyncStatus {
	return x.status
}

func (x *Store) Reconcile(id types.ID) (db.Doc, error) {

	if x.shouldReloadFromDisk(id) {
		t, err := parseID(id)
		if err != nil {
			return nil, err
		}

		filename := path.Join(x.StoragePath(), createdTimeToFileName(*t))

		//fmt.Fprintf(os.Stderr, "forcing reconcile of %d\n", int64(id))
		e, err := x.LoadFromFile(filename)
		if err != nil {
			return nil, err
		}
		return e, nil
	}

	n, err := x.loadFromCache(id)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// stat the file on disk, compare to last known mtime. if more recent
// reload
func (x *Store) loadFromCache(id types.ID) (*Note, error) {
	if e, ok := x.entries[id]; ok {
		return e, nil
	} else {
		return nil, db.ErrNoNoteFound
	}
}

func (x *Store) shouldReloadFromDisk(id types.ID) bool {
	pth := x.StoragePathDoc(id)
	finfo, err := os.Stat(pth)
	if err != nil {
		return false
	}

	if x.mtimeMap[id].Before(finfo.ModTime()) {
		return true
	}
	x.mtimeMap[id] = finfo.ModTime()

	return false
}

func (x *Store) DocType() types.DocType {
	return types.NoteDoc
}

// List satisfies the DocBackend interface
func (x *Store) List() ([]db.Doc, error) {
	l, err := x.ListAll()
	if err != nil {
		return nil, err
	}
	ret := make([]db.Doc, len(l))
	for i, e := range l {
		ret[i] = db.Doc(e)
	}
	return ret, nil
}
