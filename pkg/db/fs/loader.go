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

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/go-playground/validator"
	"gopkg.in/yaml.v3"
)

var (
	StorageFilenameFormat = "2006-01-02.md"
	StorageGlob           = "*.md"

	ErrUnableToFindMetadataSection = fmt.Errorf("unable to find metadata yaml at header of entry")
	ErrNoNextEntry                 = fmt.Errorf("no next entry found")
	ErrNoPrevEntry                 = fmt.Errorf("no previous entry found")
)

type Loader struct {
	*sync.Mutex
	Directory string        `yaml"directory" validate:"required,dir"`
	status    v1.SyncStatus `validate:"required"`
	entries   map[v1.ID]*v1.Entry
}

func New(dir string) (*Loader, error) {
	l := Loader{
		Mutex:     &sync.Mutex{},
		Directory: dir,
		status:    v1.StatusUninitialized,
		entries:   map[v1.ID]*v1.Entry{},
	}

	err := l.Validate()
	if err != nil {
		return nil, err
	}

	// Load up all the files we can find at startup
	entryFiles, err := filepath.Glob(path.Join(l.Directory, StorageGlob))
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
	fmt.Printf("all done\n")

	return &l, nil
}

func (x *Loader) Validate() error {
	validate := validator.New()
	err := validate.Struct(*x)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}

// Get loads an entry from disk and caches it in the entry map
func (x *Loader) Get(id v1.ID, forceRead bool) (*v1.Entry, error) {
	x.Lock()
	defer x.Unlock()
	e, ok := x.entries[id]
	if !ok {
		return nil, fmt.Errorf("entry %d not found", id)
	}

	if e == nil || forceRead {
		// cache not populated yet, load entry from disk
		e, err := x.LoadFromID(id)
		if err != nil {
			return nil, err
		}

		x.entries[id] = e

		return e, err
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
	t := time.Unix(int64(id), int64(0))
	fileName := t.Format(StorageFilenameFormat)
	targetpath := path.Join(x.Directory, fileName)

	return x.LoadFromFile(targetpath)
}

func (x *Loader) LoadFromReader(r io.Reader) (*v1.Entry, error) {
	var e v1.Entry

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read: %w", err)
	}

	nChunks := 3
	chunks := strings.SplitN(string(bytes), "---\n", nChunks)

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

func (x *Loader) Write(e *v1.Entry) error {
	x.status = v1.StatusSynchronizing

	targetpath := path.Join(x.Directory, e.CreationTimestamp.Format(StorageFilenameFormat))
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
	sort.Sort(v1.ByCreationTimestampEntryList(sorted))
	return sorted, nil
}

func (x *Loader) Next(e *v1.Entry) (*v1.Entry, error) {
	// TODO: this is super slow, i know. ill make it faster after PoC
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	for _, o := range elements {
		if e.ID < o.ID {
			continue
		}
		return o, nil
	}
	return nil, ErrNoNextEntry
}

func (x *Loader) Previous(e *v1.Entry) (*v1.Entry, error) {
	elements, err := x.ListAll()
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(v1.ByCreationTimestampEntryList(elements)))

	for _, o := range elements {
		if e.ID > o.ID {
			continue
		}
		return o, nil
	}
	return nil, ErrNoPrevEntry
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
