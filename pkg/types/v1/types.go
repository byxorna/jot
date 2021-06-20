package v1

import (
	"time"
)

type Entry struct {
	EntryMetadata `yaml:"metadata" validate:"required"`
	Content       string `yaml:"content" validate:""`
}

type EntryMetadata struct {
	ID                int64             `yaml:"id" validate:"required"`
	Author            string            `yaml:"author" validate:"required"`
	Title             string            `yaml:"title" validate:"required"`
	ModifiedTimestamp *time.Time        `yaml:"modifiedTimestamp,omitempty" validate:""`
	CreationTimestamp time.Time         `yaml:"creationTimestamp" validate:"required"`
	Tags              []string          `yaml:"tags" validate:"required"`
	Labels            map[string]string `yaml:"labels" validate:"required"`
}

type SyncStatus string

const (
	StatusOK            SyncStatus = "ok"
	StatusSynchronizing            = "synchronizing"
	StatusError                    = "error"
)
