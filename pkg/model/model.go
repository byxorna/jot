// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/charmbracelet/bubbles/viewport"
)

type Mode string

var (
	ViewMode Mode = "view"
	HelpMode Mode = "help"
	EditMode Mode = "edit"
	ListMode Mode = "list"

	UseHighPerformanceRendering = false
)

type Model struct {
	//db.DB // TODO: this should live in Stash now instead

	*config.Config

	UseAltScreen bool
	content      string

	Author   string
	Timeline []time.Time
	Date     time.Time
	Mode     Mode

	viewport viewport.Model

	// --- glow variables ---
	state    state
	common   *commonModel
	fatalErr error

	// Sub-model implementations
	*stashModel
	*pagerModel
}

type userMessage struct {
	// Time is when the message happened
	Time time.Time
	// Message is the terse oneline description of the issue
	Message string
	IsError bool
}

func (m *Model) Stash() Stash {
	return m.stashModel
}
