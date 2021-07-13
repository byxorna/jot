package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/adrg/xdg"
	"github.com/byxorna/jot/pkg/plugins/calendar"
	"github.com/go-playground/validator"
	"gopkg.in/yaml.v3"
)

const (
	XDGName = "jot"
)

var (
	// DefaultEntryTemplate is the default value for a new entry's content
	DefaultEntryTemplate = `- [ ] ...`

	// Default is the default configuration that is used, along with ~/.jot.yaml
	Default = Config{
		Directory:      "~/.jot.d",
		WeekendTags:    []string{"weekend"},
		WorkdayTags:    []string{"work", "$employer"},
		HolidayTags:    []string{"holiday"},
		StartWorkHours: 9 * time.Hour,
		EndWorkHours:   18*time.Hour + 30*time.Minute,
		EntryTemplate:  DefaultEntryTemplate,
		Plugins: []Plugin{
			{Name: calendar.PluginName},
		},
	}
)

type Config struct {
	Directory      string        `yaml:"directory" validate:"required"`
	WeekendTags    []string      `yaml:"weekendTags" validate:"unique"`
	WorkdayTags    []string      `yaml:"workdayTags" validate:"unique"`
	HolidayTags    []string      `yaml:"holidayTags" validate:"unique"`
	StartWorkHours time.Duration `yaml:"startWorkHours" validate:"required"`
	EndWorkHours   time.Duration `yaml:"endWorkHours" validate:"required"`
	Plugins        []Plugin      `yaml:"plugins" validate:"unique"`
	EntryTemplate  string        `yaml:"entry_template" validate:""`
}

type Plugin struct {
	Name string `yaml:"name" validate:"required"`
}

func NewFromReader(r io.Reader) (*Config, error) {
	c := Default

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read Config: %w", err)
	}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal Config: %w", err)
	}

	validate := validator.New()
	err = validate.Struct(c)
	if err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &c, nil
}

func RuntimeFile(filename string) (string, error) {
	return xdg.RuntimeFile(fmt.Sprintf("%s/%s", XDGName, filename))
}
