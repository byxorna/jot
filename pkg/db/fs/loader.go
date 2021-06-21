package fs

import (
	"time"

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/go-playground/validator"
)

var ()

type FSLoader struct {
	Directory string `yaml"directory" validate:"required,dir"`
}

func (x *FSLoader) Validate() error {
	validate := validator.New()
	err := validate.Struct(*x)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}

func (x *FSLoader) Create(tags []string, labels map[string]string) (*v1.Entry, error) {
	t := time.Now()
	e := v1.Entry{
		Content: "",
		EntryMetadata: v1.EntryMetadata{
			ID:                t.Unix(),
			Title:             "", // default empty
			Tags:              tags,
			Labels:            labels,
			CreationTimestamp: time.Now(),
		},
	}
}

func (x *FSLoader) Update(e *v1.Entry) (*v1.Entry, error) {
}

func (x *FSLoader) List() ([]*v1.Entry, error) {
}

func (x *FSLoader) Status() v1.SyncStatus {
	return v1.StatusOK
}

func (x *FSLoader) ParseHeader(header string) (*v1.EntryMetadata, bool) {
	fallback := &v1.EntryMetadata{}
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

func (x *FSLoader) Load() error {
	var content string
	var err error

	if m.FileName != "" {
		content, err = readFile(m.FileName)
	} else {
		content, err = readStdin()
	}

	if err != nil {
		return err
	}

	content = strings.ReplaceAll(content, altDelimiter, delimiter)
	slides := strings.Split(content, delimiter)

	metaData, exists := meta.New().ParseHeader(slides[0])
	// If the user specifies a custom configuration options
	// skip the first "slide" since this is all configuration
	if exists {
		slides = slides[1:]
	}

	m.Slides = slides
	if m.Theme == nil {
		m.Theme = styles.SelectTheme(metaData.Theme)
	}

	return nil
}
