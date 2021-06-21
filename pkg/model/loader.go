package model

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/go-playground/validator"
	"gopkg.in/yaml.v3"
)

func NewFromConfigFile(path string, user string) (*Model, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	c := v1.Config{}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	err = validate.Struct(c)
	if err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	m := Model{
		Config: c,
		Author: user,
		Date:   time.Now(), //.Format("2006-01-02"),
	}

	err = m.initBackend()
	if err != nil {
		return nil, fmt.Errorf("error initializing storage provider: %w", err)
	}
	return &m, nil
}
