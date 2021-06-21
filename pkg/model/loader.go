package model

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

var (
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

	// Open either the appropriate entry for today, or create a new one
	if entries, err := m.DB.ListAll(); err == nil {
		if len(entries) == 0 || entries[0].CreationTimestamp.Format(fs.StorageFilenameFormat) != m.Date.Format(fs.StorageFilenameFormat) {
			title := m.Date.Format("2006-01-02")
			e, err := m.DB.CreateOrUpdateEntry(&v1.Entry{
				EntryMetadata: v1.EntryMetadata{
					Author: m.Author,
					Title:  title,
				},
				Content: fmt.Sprintf("# %s\n\n", title),
			})
			if err != nil {
				return nil, fmt.Errorf("unable to create new entry: %w", err)
			}
			m.Entry = e
		}
	}

	return &m, nil
}
