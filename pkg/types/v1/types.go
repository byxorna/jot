package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/types"
	"github.com/go-playground/validator"
	"github.com/voicera/gooseberry/urn"
)

type ID int64

type Note struct {
	Metadata NoteMetadata `yaml:"metadata" validate:"required"`
	Content  string       `yaml:"content" validate:""`
}

type NoteMetadata struct {
	ID     ID     `yaml:"id" validate:"required"`
	Author string `yaml:"author" validate:"required"`
	Title  string `yaml:"title,omitempty" validate:""`
	///ModifiedTimestamp time.Time         `yaml:"modified,omitempty" validate:""`
	CreationTimestamp time.Time         `yaml:"created" validate:"required"`
	Tags              []string          `yaml:"tags,omitempty,flow" validate:""`
	Labels            map[string]string `yaml:"labels,omitempty,flow" validate:""`
}

type SyncStatus string

const (
	StatusUninitialized SyncStatus = "uninitialized"
	StatusOK            SyncStatus = "ok"
	StatusOffline       SyncStatus = "offline"
	StatusSynchronizing SyncStatus = "synchronizing"
	StatusError         SyncStatus = "error"
)

type ByID []ID

func (p ByID) Len() int {
	return len(p)
}

func (p ByID) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p ByID) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type ByCreationTimestampNoteList []*Note

func (p ByCreationTimestampNoteList) Len() int {
	return len(p)
}

func (p ByCreationTimestampNoteList) Less(i, j int) bool {
	return p[i].Metadata.CreationTimestamp.Before(p[j].Metadata.CreationTimestamp)
}

func (p ByCreationTimestampNoteList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (e *Note) Validate() error {
	validate := validator.New()
	err := validate.Struct(*e)
	//validationErrors := err.(validator.ValidationErrors)
	return err
}

func (e *Note) MatchesFilter(needle string) bool {
	return strings.Contains(e.Content, needle)
}

func (e *Note) Identifier() string {
	u := urn.NewURN(e.DocType().String(), fmt.Sprintf("%d", e.Metadata.ID))
	return u.String()
}

func (e *Note) DocType() types.DocType {
	return types.NoteDoc
}

func (e *Note) SelectorLabels() map[string]string {
	return e.Metadata.Labels
}

func (e *Note) SelectorTags() []string {
	return e.Metadata.Tags
}
