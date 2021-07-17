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
	"github.com/byxorna/jot/pkg/net/http"
	"github.com/byxorna/jot/pkg/plugins/calendar"
	"github.com/byxorna/jot/pkg/plugins/keep"
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

	pluginAuthScopes := []string{}
	for _, sec := range configuration.Sections {
		switch sec.Plugin {
		case config.PluginTypeKeep:
			pluginAuthScopes = append(pluginAuthScopes, keep.GoogleAuthScopes...)
		case config.PluginTypeCalendar:
			pluginAuthScopes = append(pluginAuthScopes, calendar.GoogleAuthScopes...)
		}
	}

	client, err := http.NewClientWithGoogleAuthedScopes(ctx, pluginAuthScopes...)

	common := commonModel{}
	stashModel, err := newStashModel(&common, client, &configuration)
	if err != nil {
		return nil, err
	}
	pagerModel := newPagerModel(&common)

	m := Model{
		UseAltScreen: useAltScreen,
		Config:       &configuration,
		Author:       user,
		Date:         time.Now(),
		Mode:         ViewMode,

		// glow bits
		common:     &common,
		state:      stateShowStash,
		pagerModel: pagerModel,
		stashModel: stashModel,
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
