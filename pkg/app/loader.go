package app

import (
	"os"

	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/mitchellh/go-homedir"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/lipgloss"
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

	err = m.initModel(ctx)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (a *Application) initModel(ctx context.Context) error {
	// collect all enabled plugin auth scopes when we create our http client
	//authScopes := []string{}
	//for _, sec := range a.Config.Sections {
	//	switch sec.Plugin {
	//	case config.PluginTypeCalendar:
	//		authScopes = append(authScopes, calendar.GoogleAuthScopes...)
	//	case config.PluginTypeKeep:
	//		authScopes = append(authScopes, keep.GoogleAuthScopes...)
	//	}
	//}

	//client, err := http.NewDefaultClient(ctx, authScopes...)
	//if err != nil {
	//	return fmt.Errorf("failed to create client for auth scopes %v: %w", strings.Join(authScopes, ","), err)
	//}

	//var s []*section
	//for _, sec := range a.Config.Sections {
	//	switch sec.Plugin {

	//	case config.PluginTypeNotes:
	//		noteBackend, err := fs.New(a.Config.Directory, true)
	//		if err != nil {
	//			return fmt.Errorf("error initializing storage provider: %w", err)
	//		}

	//		notes := newSectionModel(sec.Name, noteBackend)
	//		s = append(s, &notes)
	//		fsPlugin = noteBackend

	//	case config.PluginTypeCalendar:
	//		/*
	//			client, err := http.NewDefaultClient(calendar.GoogleAuthScopes...)
	//			if err != nil {
	//				return nil, fmt.Errorf("%s failed to create client: %w", sec.Plugin, err)
	//			}
	//		*/
	//		cp, err := calendar.New(ctx, client, sec.Settings, sec.Features)
	//		if err != nil {
	//			return fmt.Errorf("%s failed to initialize: %w", sec.Plugin, err)
	//		}
	//		today := newSectionModel(sec.Name, cp)
	//		s = append(s, &today)

	//	case config.PluginTypeKeep:
	//		//client, err := http.NewClientWithGoogleAuthedScopes(context.TODO(), sec.Plugin, keep.GoogleAuthScopes...)
	//		/*
	//			client, err := http.NewDefaultClient(keep.GoogleAuthScopes...)
	//			if err != nil {
	//				return nil, fmt.Errorf("%s failed to create client: %w", sec.Plugin, err)
	//			}
	//		*/

	//		kp, err := keep.New(ctx, client)
	//		if err != nil {
	//			return fmt.Errorf("%s failed to initialize: %w", sec.Plugin, err)
	//		}
	//		keepClient := newSectionModel(sec.Name, kp)
	//		s = append(s, &keepClient)

	//	default:
	//		// TODO: maybe skip initialization? :thinking:
	//		return fmt.Errorf("unsupported plugin %v for section name %s", sec.Plugin, sec.Name)
	//	}
	//}

	return nil
}
