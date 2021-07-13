package model

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/mitchellh/go-homedir"
)

var (
	// CreateDirectoryIfMissing creates config.Directory if not already existing
	CreateDirectoryIfMissing = true
)

func NewFromConfigFile(path string, user string, useAltScreen bool) (*Model, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	var configuration config.Config
	f, err := os.Open(expandedPath)
	if err != nil && f == nil {
		// if the file is missing, ignore and use the default config
		configuration = config.Default
	} else {
		cfg, err := config.NewFromReader(f)
		if err != nil {
			return nil, fmt.Errorf("unable to load configuration: %w", err)
		}
		configuration = *cfg
	}

	common := commonModel{}

	m := Model{
		UseAltScreen: useAltScreen,
		Config:       configuration,
		Author:       user,
		Date:         time.Now(),
		Mode:         ViewMode,

		// glow bits
		common: &common,
		state:  stateShowStash,
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
				Content: m.Config.EntryTemplate,
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
