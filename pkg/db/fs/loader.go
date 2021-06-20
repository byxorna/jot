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
