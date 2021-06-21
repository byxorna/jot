package model

import (
	"fmt"
	"io/ioutil"

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
	}

	// TODO: switch here on backend type and load appropriate db provider
	loader, err := fs.New(m.Config.Directory)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage provider: %w", err)
	}
	m.DB = loader

	if m.DB.HasEntry(v1.ID(m.Date.Unix())) {
		e, err := m.DB.Get(v1.ID(m.Date.Unix()), false)
		if err != nil {
			return nil, err
		}
		m.Entry = e
	} else {
		e, err := m.DB.CreateOrUpdateEntry(&v1.Entry{
			EntryMetadata: v1.EntryMetadata{},
			Content:       ``,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to create new entry: %w", err)
		}
		m.Entry = e
	}

	return &m, nil
}
