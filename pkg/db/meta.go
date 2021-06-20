// Package meta implements markdown frontmatter parsing for simple
// slides configuration
// Source: https://github.com/maaslalani/slides/blob/main/internal/meta/meta.go
package meta

import (
	"strings"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Meta contains all of the data to be parsed
// out of a markdown file's header section
type Meta struct {
	Theme string `yaml:"theme"`
}

func New() *Meta {
	return &Meta{}
}

// ParseHeader parses metadata from a slideshows header slide
// including theme information
//
// If no front matter is provided, it will fallback to the default theme and
// return false to acknowledge that there is no front matter in this slide
func (m *Meta) ParseHeader(header string) (*Meta, bool) {
	fallback := &Meta{Theme: "default"}
	bytes, err := frontmatter.Parse(strings.NewReader(header), &m)
	if err != nil {
		return fallback, false
	}

	err = yaml.Unmarshal(bytes, &m)
	if err != nil {
		return fallback, false
	}

	return m, true
}
