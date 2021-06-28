package v1

import (
	"time"

	"github.com/go-playground/validator"
)

type ID int64

type Entry struct {
	EntryMetadata `yaml:"metadata" validate:"required"`
	Content       string `yaml:"content" validate:""`
}

type EntryMetadata struct {
	ID     ID     `yaml:"id" validate:"required"`
	Author string `yaml:"author" validate:"required"`
	Title  string `yaml:"title,omitempty" validate:""`
	///ModifiedTimestamp time.Time         `yaml:"modified,omitempty" validate:""`
	CreationTimestamp time.Time         `yaml:"created" validate:"required"`
	Tags              []string          `yaml:"tags,omitempty" validate:""`
	Labels            map[string]string `yaml:"labels,omitempty" validate:""`
}

type SyncStatus string

const (
	StatusUninitialized SyncStatus = "uninitialized"
	StatusOK            SyncStatus = "ok"
	StatusOffline       SyncStatus = "offline"
	StatusSynchronizing SyncStatus = "synchronizing"
	StatusError         SyncStatus = "error"
)

type Config struct {
	Directory string `yaml:"directory" validate:"required"`
}

type ByCreationTimestampEntryList []*Entry
type ByModifiedTimestampEntryList []*Entry

func (p ByCreationTimestampEntryList) Len() int {
	return len(p)
}

func (p ByCreationTimestampEntryList) Less(i, j int) bool {
	return p[i].CreationTimestamp.Before(p[j].CreationTimestamp)
}

func (p ByCreationTimestampEntryList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (e *Entry) Validate() error {
	validate := validator.New()
	err := validate.Struct(*e)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}
