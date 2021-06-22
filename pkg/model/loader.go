package model

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

var (
	EntryTemplate = `- [ ] ...`
	DefaultConfig = v1.Config{
		//Directory: "~/.jot.d",
		Directory: "test/notes",
	}
)

func NewFromConfigFile(path string, user string) (*Model, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	c := v1.Config{}
	bytes, err := ioutil.ReadFile(expandedPath)
	if err != nil {
		// ignore, just use default config
		c = DefaultConfig
	} else {
		err = yaml.Unmarshal(bytes, &c)
		if err != nil {
			return nil, err
		}
	}

	validate := validator.New()
	err = validate.Struct(c)
	if err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	m := Model{
		Config: c,
		Author: user,
		Date:   time.Now(),
	}

	// TODO: switch here on backend type and load appropriate db provider
	loader, err := fs.New(m.Config.Directory)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage provider: %w", err)
	}
	m.DB = loader
	fmt.Printf("loaded %d entries\n", m.DB.Count())

	// Open either the appropriate entry for today, or create a new one
	if entries, err := m.DB.ListAll(); err == nil {
		// if the most recent entry isnt the same as our expected filename, create a new entry for today
		if len(entries) == 0 || len(entries) > 0 && entries[0].CreationTimestamp.Format(fs.StorageFilenameFormat) != m.Date.Format(fs.StorageFilenameFormat) {
			title := TitleFromTime(m.Date)
			e, err := m.DB.CreateOrUpdateEntry(&v1.Entry{
				EntryMetadata: v1.EntryMetadata{
					Author: m.Author,
					Title:  title,
				},
				Content: fmt.Sprintf("# %s\n\n%s", title, EntryTemplate),
			})
			if err != nil {
				return nil, fmt.Errorf("unable to create new entry: %w", err)
			}
			m.Entry = e
		} else {
			// just grab the first entry
			m.Entry = entries[0]
		}
	}

	return &m, nil
}

func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		return "", fmt.Errorf("No entry found")
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			return "", err
		}
	}

	return b.String(), nil
}
