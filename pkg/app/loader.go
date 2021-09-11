package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/go-homedir"
)

var (
	// CreateDirectoryIfMissing creates config.Directory if not already existing
	CreateDirectoryIfMissing = true
)

func New(ctx context.Context, path string, user string, useAltScreen bool) (*Application, error) {
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

	m := Application{
		keys:       keys,
		help:       help.NewModel(),
		inputStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),

		UseAltScreen: useAltScreen,
		Config:       &configuration,
	}

	err = m.initPlugins(ctx)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (a *Application) initPlugins(ctx context.Context) error {
	client := http.DefaultClient
	//client, err := http.NewDefaultClient(ctx) //authScopes...)
	//if err != nil {
	//	return fmt.Errorf("failed to create client for auth scopes %v: %w", strings.Join(authScopes, ","), err)
	//}

	var plugins []*section
	for _, sec := range a.Config.Sections {
		switch sec.Plugin {

		case config.PluginTypeNotes:
			noteBackend, err := fs.New(a.Config.Directory, true)
			if err != nil {
				return fmt.Errorf("error initializing storage provider: %w", err)
			}

			notes := newSectionModel(sec.Name, noteBackend)
			s = append(s, &notes)
	}

  delegateKeys :=newDelegateKeyMap()
  listKeys
  delegate := newDe
  pluginsList := list.NewModel(plugins, delegate, 0, 0)

	return nil
}
