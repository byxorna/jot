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
	"github.com/byxorna/jot/pkg/model/document"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

var (
	StorageFilenameFormat = "2006-01-02.md"
	StorageGlob           = "*.md"

	ErrUnableToFindMetadataSection = fmt.Errorf("unable to find metadata yaml at header of note")
)

type Store struct {
	*sync.Mutex

	Directory string `yaml"directory" validate:"required,dir"`

	status   v1.SyncStatus `validate:"required"`
	entries  map[v1.ID]*v1.Note
	mtimeMap map[v1.ID]time.Time
	watcher  *fsnotify.Watcher
}

func New(dir string, createDirIfMissing bool) (*Store, error) {
	expandedPath, err := homedir.Expand(dir)
	if err != nil {
		return nil, err
	}

	s := Store{
		Mutex:     &sync.Mutex{},
		Directory: expandedPath,
		status:    v1.StatusUninitialized,
		entries:   map[v1.ID]*v1.Note{},
		mtimeMap:  map[v1.ID]time.Time{},
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
				return nil, err
			}
			s.entries[e.ID] = e
		}
	}

	if err := s.startWatcher(); err != nil {
		return nil, fmt.Errorf("unable to watch %s: %w", s.Directory, err)
	}

	s.status = v1.StatusOK

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

				//fmt.Fprintf(os.Stderr, "event: %v\n", event)
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					//fmt.Fprintln(os.Stderr, "modified file:", event.Name)
					entries, _ := x.ListAll()
					for _, e := range entries {
						expectedFileName := path.Base(x.StoragePath(e.ID))
						if expectedFileName == path.Base(event.Name) {
							//fmt.Fprintf(os.Stderr, "reconciling %s\n", event.Name)
							_, err := x.Reconcile(e.ID)
							if err != nil {
								// TODO: do something better
								fmt.Fprintf(os.Stderr, "error reconciling %d: %v\n", int64(e.ID), err)
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

// Get loads an note from disk and caches it in the note map
func (x *Store) Get(id v1.ID, forceRead bool) (*v1.Note, error) {
	if forceRead {
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

func (x *Store) CreateOrUpdateNote(e *v1.Note) (*v1.Note, error) {
	x.Lock()
	defer x.Unlock()

	if e.CreationTimestamp.IsZero() {
		e.CreationTimestamp = time.Now()
	}

	if e.ID == 0 {
		e.ID = v1.ID(e.CreationTimestamp.Unix())
	}

	//if x.HasNote(e.ID) {
	//	t := time.Now()
	//	e.Note.ModifiedTimestamp = &t
	//}

	// TODO: union tags and labels with defaults

	if err := x.Write(e); err != nil {
		return nil, fmt.Errorf("unable to store note %d: %w", e.ID, err)
	}

	x.entries[e.ID] = e

	return e, nil
}

func (x *Store) LoadFromFile(fileName string) (*v1.Note, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", fileName, err)
	}

	return x.LoadFromReader(f)
}

func (x *Store) LoadFromID(id v1.ID) (*v1.Note, error) {
	return x.LoadFromFile(x.StoragePath(id))
}

func (x *Store) LoadFromReader(r io.Reader) (*v1.Note, error) {
	var e v1.Note

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read: %w", err)
	}

	nChunks := 3
	chunks := strings.SplitN(string(bytes), "---", nChunks)

	if len(chunks) != nChunks {
		return nil, fmt.Errorf("unable to parse metadata section: %w", ErrUnableToFindMetadataSection)
	}

	err = yaml.Unmarshal([]byte(chunks[1]), &e.NoteMetadata)
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

func (x *Store) expandedStoragePath(id v1.ID) string {
	expandedPath, _ := homedir.Expand(x.shortStoragePath(id))
	return expandedPath
}

func (x *Store) StoragePath(id v1.ID) string {
	return x.expandedStoragePath(id)
}

func (x *Store) shortStoragePath(id v1.ID) string {
	t := time.Unix(int64(id), int64(0)).UTC()
	fullPath := path.Join(x.Directory, t.Format(StorageFilenameFormat))
	return fullPath
}

func (x *Store) Write(e *v1.Note) error {
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

	metadata, err := yaml.Marshal(e.NoteMetadata)
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to marshal note metadata for %d: %w", e.ID, err)
	}

	_, err = f.WriteString(fmt.Sprintf("---\n%s\n---\n", metadata))
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to write note metadata for %d: %w", e.ID, err)
	}

	_, err = f.WriteString(e.Content + "\n")
	if err != nil {
		x.status = v1.StatusError
		return fmt.Errorf("unable to write note %d: %w", e.ID, err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("unable to sync note %d: %w", e.ID, err)
	}

	x.status = v1.StatusOK
	return nil
}

// ListAll returns entries in newest to oldest order
func (x *Store) ListAll() ([]*v1.Note, error) {
	x.Lock()
	defer x.Unlock()

	sorted := []*v1.Note{}
	for _, e := range x.entries {
		sorted = append(sorted, e)
	}
	sort.Sort(sort.Reverse(v1.ByCreationTimestampNoteList(sorted)))
	return sorted, nil
}

func (x *Store) idx(list []*v1.Note, id v1.ID) (int, error) {

	for i, o := range list {
		if id == o.ID {
			return i, nil
		}
	}
	return 0, db.ErrNoNoteFound
}

func (x *Store) Next(id v1.ID) (*v1.Note, error) {
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

func (x *Store) Previous(id v1.ID) (*v1.Note, error) {
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

func (x *Store) HasNote(id v1.ID) bool {
	_, ok := x.entries[id]
	return ok
}

func (x *Store) Status() v1.SyncStatus {
	return x.status
}

func (x *Store) Reconcile(id v1.ID) (*v1.Note, error) {
	// stat the file on disk, compare to last known mtime. if more recent
	// reload
	if !x.HasNote(id) || x.ShouldReloadFromDisk(id) {
		//fmt.Fprintf(os.Stderr, "forcing reconcile of %d\n", int64(id))
		e, err := x.LoadFromID(id)
		if err != nil {
			return nil, err
		}
		return e, nil
	}

	if e, ok := x.entries[id]; ok {
		return e, nil
	} else {
		return nil, db.ErrNoNoteFound
	}
}

func (x *Store) ShouldReloadFromDisk(id v1.ID) bool {
	finfo, err := os.Stat(x.StoragePath(id))
	if err != nil {
		return false
	}

	if x.mtimeMap[id].Before(finfo.ModTime()) {
		return true
	}
	x.mtimeMap[id] = finfo.ModTime()

	return false
}

func (x *Store) DocTypes() document.DocTypeSet {
	return document.NewDocTypeSet(document.NoteDoc)
}

func (x *Store) List() []... {
  // TODO: need to return []Doc but its unclear how the heirarchy works here


}





