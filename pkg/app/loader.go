package app

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/bubbles/help"
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
		help: help.NewModel(),

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

	//var plugins []list.Item
	var plugins []*Plugin
	for _, sec := range a.Config.Sections {
		switch sec.Plugin {

		case config.PluginTypeNotes:
			plugins = append(plugins, &Plugin{name: sec.Name})
		default:
			log.Printf("ignoring %s plugin", sec.Name)
		}
	}
	plugins = append(plugins, &Plugin{name: "Test"})
	plugins = append(plugins, &Plugin{name: "Test 2"})
	plugins = append(plugins, &Plugin{name: "Test 3"})
	a.plugins = plugins

	delegateKeys := newDelegateKeyMap()
	delegate := newSectionDelegate(delegateKeys)

	listItems := itemsFromPlugins(plugins)
	a.list = list.NewModel(listItems, delegate, 0, 0)

	return nil
}

func itemsFromPlugins(plugins []*Plugin) []list.Item {
	lx := make([]list.Item, len(plugins))
	for i, _ := range plugins {
		lx[i] = plugins[i]
	}
	return lx
}
