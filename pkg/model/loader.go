package model

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/plugins/calendar"
	"github.com/mitchellh/go-homedir"
)

var (
	// CreateDirectoryIfMissing creates config.Directory if not already existing
	CreateDirectoryIfMissing = true

	calendarPlugin *calendar.Client
)

func NewFromConfigFile(ctx context.Context, path string, user string, useAltScreen bool) (*Model, error) {
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

	// enable plugins
	for _, section := range m.Config.Sections {
		fmt.Printf("enabling plugin for %s section\n", section.Type)
		switch section.Type {
		case config.SectionTypeCalendar:
			cp, err := calendar.New(ctx)
			if err != nil {
				return nil, fmt.Errorf("%s failed to initialize: %w", section.Type, err)
			}
			calendarPlugin = cp
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
