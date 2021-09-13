package app

import (
	"context"
	"fmt"
	"os"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/plugins/filecommander"
	//"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
		keys: DefaultKeyMap(),
		//help: help.NewModel(),

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

	a.plugins = []Plugin{}
	for _, sec := range a.Config.Sections {
		switch sec.Plugin {

		//	case config.PluginTypeNotes:
		//		notes, err := fs.New(a.Config.Directory, true)
		//		if err != nil {
		//			return err
		//		}
		//		plugins = append(plugins, &Plugin{name: sec.Name, DocBackend: notes})
		case config.PluginTypeFileCommander:
			fc, err := filecommander.New(a.Config.Directory)
			if err != nil {
				return err
			}
			a.plugins = append(a.plugins, fc)
		default:
			fmt.Printf("ignoring %s plugin\n", sec.Name)
		}
	}

	delegate := newSectionDelegate()
	items := []list.Item{}
	for _, p := range a.plugins {
		items = append(items, item{title: p.Name(), desc: p.Description()})
	}

	for _, x := range []string{"test 1", "test 2", "test 3"} {
		items = append(items, item{title: x, desc: x})
	}
	a.list = list.NewModel(items, delegate, 0, 0)
	a.list.Title = "Plugins"
	return nil
}
