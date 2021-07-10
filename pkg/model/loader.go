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
	// EntryTemplate is the default value for a new entry's content
	EntryTemplate = `- [ ] ...`

	// DefaultConfig is the default configuration that is used, along with ~/.jot.yaml
	DefaultConfig = v1.Config{
		Directory:      "~/.jot.d",
		WeekendTags:    []string{"weekend"},
		WorkdayTags:    []string{"work", "$employer"},
		HolidayTags:    []string{"holiday"},
		StartWorkHours: 9 * time.Hour,
		EndWorkHours:   18*time.Hour + 30*time.Minute,
	}

	// CreateDirectoryIfMissing creates config.Directory if not already existing
	CreateDirectoryIfMissing = true
)

func NewFromConfigFile(path string, user string, useAltScreen bool) (*Model, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	c := DefaultConfig
	bytes, err := ioutil.ReadFile(expandedPath)
	if err == nil {
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

	common := commonModel{}

	m := Model{
		UseAltScreen: useAltScreen,
		Config:       c,
		Author:       user,
		Date:         time.Now(),
		Mode:         ViewMode,

		// glow bits
		common: &common,
		state:  stateFocusStashList,
		pager:  newPagerModel(&common),
		stash:  newStashModel(&common),
	}

	// TODO: switch here on backend type and load appropriate db provider
	loader, err := fs.New(m.Config.Directory, true)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage provider: %w", err)
	}
	m.DB = loader
	fmt.Printf("loaded %d entries\n", m.DB.Count())

	// Open either the appropriate entry for today, or create a new one
	if entries, err := m.DB.ListAll(); err == nil {
		// if the most recent entry isnt the same as our expected filename, create a new entry for today
		if len(entries) == 0 || len(entries) > 0 && entries[0].CreationTimestamp.Format(fs.StorageFilenameFormat) != m.Date.Format(fs.StorageFilenameFormat) {
			_, err := m.DB.CreateOrUpdateEntry(&v1.Entry{
				EntryMetadata: v1.EntryMetadata{
					Author: m.Author,
					Title:  m.TitleFromTime(m.Date),
					Tags:   m.DefaultTagsForTime(m.Date),
				},
				Content: EntryTemplate,
			})
			if err != nil {
				return nil, fmt.Errorf("unable to create new entry: %w", err)
			}
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
