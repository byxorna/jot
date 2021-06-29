// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/ui.go
package model

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/gitcha"
)

const (
	noteCharacterLimit   = 256             // should match server
	statusMessageTimeout = time.Second * 2 // how long to show status messages like "stashed!"
	ellipsis             = "…"
)

var (
	glowLogoTextColor = lib.Color("#ECFD65")

	markdownExtensions = []string{
		"*.md", "*.mdown", "*.mkdn", "*.mkd", "*.markdown",
	}

	// True if we're logging to a file, in which case we'll log more stuff.
	debug = false
)

// NewProgram returns a new Tea program.
func NewProgram() *tea.Program {
	return tea.NewProgram(newModel())
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type initLocalFileSearchMsg struct {
	cwd string
	ch  chan gitcha.SearchResult
}
type foundLocalFileMsg gitcha.SearchResult
type localFileSearchFinished struct{}
type gotStashMsg []*v1.Entry
type stashLoadErrMsg struct{ err error }
type statusMessageTimeoutMsg applicationContext
type stashSuccessMsg markdown
type stashFailMsg struct {
	err      error
	markdown markdown
}

// applicationContext indicates the area of the application something appies
// to. Occasionally used as an argument to commands and messages.
type applicationContext int

const (
	stashContext applicationContext = iota
	pagerContext
)

// state is the top-level application state.
type state int

const (
	stateShowStash state = iota
	stateShowDocument
)

func (s state) String() string {
	return map[state]string{
		stateShowStash:    "showing file listing",
		stateShowDocument: "showing document",
	}[s]
}

// Common stuff we'll need to access in all models.
type commonModel struct {
	cwd    string
	width  int
	height int
}

type model struct {
	common   *commonModel
	state    state
	fatalErr error

	// Sub-models
	stash stashModel
	pager pagerModel

	// Channel that receives paths to local markdown files
	// (via the github.com/muesli/gitcha package)
	localFileFinder chan gitcha.SearchResult
}

// unloadDocument unloads a document from the pager. Note that while this
// method alters the model we also need to send along any commands returned.
func (m *model) unloadDocument() []tea.Cmd {
	m.state = stateShowStash
	m.stash.viewState = stashStateReady
	m.pager.unload()
	m.pager.showHelp = false

	var batch []tea.Cmd
	if m.pager.viewport.HighPerformanceRendering {
		batch = append(batch, tea.ClearScrollArea)
	}

	if !m.stash.loadingDone() {
		batch = append(batch, spinner.Tick)
	}
	return batch
}

func newModel() tea.Model {
	/*
		if cfg.GlamourStyle == "auto" {
			if te.HasDarkBackground() {
				cfg.GlamourStyle = "dark"
			} else {
				cfg.GlamourStyle = "light"
			}
		}

		if len(cfg.DocumentTypes) == 0 {
			cfg.DocumentTypes.Add(LocalDoc, StashedDoc)
		}
	*/

	common := commonModel{}

	return model{
		common: &common,
		state:  stateShowStash,
		pager:  newPagerModel(&common),
		stash:  newStashModel(&common),
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd

	//if d.Contains(StashedDoc) {
	cmds = append(cmds, spinner.Tick)
	//}

	//if d.Contains(LocalDoc) {
	cmds = append(cmds, findLocalFiles(m))
	//}

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If there's been an error, any key exits
	if m.fatalErr != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.state == stateShowDocument {
				batch := m.unloadDocument()
				return m, tea.Batch(batch...)
			}

		case "q":
			var cmd tea.Cmd

			switch m.state {
			case stateShowStash:
				// pass through all keys if we're editing the filter
				if m.stash.filterState == filtering || m.stash.selectionState == selectionSettingNote {
					m.stash, cmd = m.stash.update(msg)
					return m, cmd
				}

			// Special cases for the pager
			case stateShowDocument:
				switch m.pager.state {
				// If setting a note send all keys straight through
				case pagerStateSetNote:
					var batch []tea.Cmd
					newPagerModel, cmd := m.pager.update(msg)
					m.pager = newPagerModel
					batch = append(batch, cmd)
					return m, tea.Batch(batch...)
				}
			}

			return m, tea.Quit

		case "left", "h", "delete":
			if m.state == stateShowDocument && m.pager.state != pagerStateSetNote {
				cmds = append(cmds, m.unloadDocument()...)
				return m, tea.Batch(cmds...)
			}

		// Ctrl+C always quits no matter where in the application you are.
		case "ctrl+c":
			return m, tea.Quit
		}

	// Window size is received when starting up and on every resize
	case tea.WindowSizeMsg:
		m.common.width = msg.Width
		m.common.height = msg.Height
		m.stash.setSize(msg.Width, msg.Height)
		m.pager.setSize(msg.Width, msg.Height)

	case initLocalFileSearchMsg:
		m.localFileFinder = msg.ch
		m.common.cwd = msg.cwd
		cmds = append(cmds, findNextLocalFile(m))

	case stashLoadErrMsg:
		m.common.authStatus = authFailed

	case fetchedMarkdownMsg:
		// We've loaded a markdown file's contents for rendering
		m.pager.currentDocument = *msg
		msg.Body = string(utils.RemoveFrontmatter([]byte(msg.Body)))
		cmds = append(cmds, renderWithGlamour(m.pager, msg.Body))

	case contentRenderedMsg:
		m.state = stateShowDocument

	case noteSavedMsg:
		// A note was saved to a document. This will have been done in the
		// pager, so we'll need to find the corresponding note in the stash.
		// So, pass the message to the stash for processing.
		stashModel, cmd := m.stash.update(msg)
		m.stash = stashModel
		return m, cmd

	case localFileSearchFinished, gotStashMsg, gotNewsMsg:
		// Always pass these messages to the stash so we can keep it updated
		// about network activity, even if the user isn't currently viewing
		// the stash.
		stashModel, cmd := m.stash.update(msg)
		m.stash = stashModel
		return m, cmd

	case foundLocalFileMsg:
		newMd := localFileToMarkdown(m.common.cwd, gitcha.SearchResult(msg))
		m.stash.addMarkdowns(newMd)
		if m.stash.filterApplied() {
			newMd.buildFilterValue()
		}
		if m.stash.shouldUpdateFilter() {
			cmds = append(cmds, filterMarkdowns(m.stash))
		}
		cmds = append(cmds, findNextLocalFile(m))

	case stashSuccessMsg:
		// Common handling that should happen regardless of application state
		md := markdown(msg)
		m.stash.addMarkdowns(&md)
		m.common.filesStashed[msg.stashID] = struct{}{}
		delete(m.common.filesStashing, md.stashID)

		if m.stash.filterApplied() {
			for _, v := range m.stash.filteredMarkdowns {
				if v.stashID == msg.stashID && v.docType == ConvertedDoc {
					// Add the server-side ID we got back so we can do things
					// like rename and stash it.
					v.ID = msg.ID

					// Keep the unique ID in sync so we can do things like
					// delete. Note that the markdown received a new unique ID
					// when it was added to the file listing in
					// stash.addMarkdowns.
					v.uniqueID = md.uniqueID
					break
				}
			}
		}

	case stashFailMsg:
		// Common handling that should happen regardless of application state
		delete(m.common.filesStashed, msg.markdown.stashID)
		delete(m.common.filesStashing, msg.markdown.stashID)

	case filteredMarkdownMsg:
		if m.state == stateShowDocument {
			newStashModel, cmd := m.stash.update(msg)
			m.stash = newStashModel
			cmds = append(cmds, cmd)
		}
	}

	// Process children
	switch m.state {
	case stateShowStash:
		newStashModel, cmd := m.stash.update(msg)
		m.stash = newStashModel
		cmds = append(cmds, cmd)

	case stateShowDocument:
		newPagerModel, cmd := m.pager.update(msg)
		m.pager = newPagerModel
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.fatalErr != nil {
		return errorView(m.fatalErr, true)
	}

	switch m.state {
	case stateShowDocument:
		return m.pager.View()
	default:
		return m.stash.view()
	}
}

// COMMANDS

func findLocalFiles(m model) tea.Cmd {
	return func() tea.Msg {
		var (
			cwd = m.common.cfg.WorkingDirectory
			err error
		)

		if cwd == "" {
			cwd, err = os.Getwd()
		} else {
			var info os.FileInfo
			info, err = os.Stat(cwd)
			if err == nil && info.IsDir() {
				cwd, err = filepath.Abs(cwd)
			}
		}

		// Note that this is one error check for both cases above
		if err != nil {
			if debug {
				log.Println("error finding local files:", err)
			}
			return errMsg{err}
		}

		if debug {
			log.Println("local directory is:", cwd)
		}

		var ignore []string
		if !m.common.cfg.ShowAllFiles {
			ignore = ignorePatterns(m)
		}

		ch, err := gitcha.FindFilesExcept(cwd, markdownExtensions, ignore)
		if err != nil {
			if debug {
				log.Println("error finding local files:", err)
			}
			return errMsg{err}
		}

		return initLocalFileSearchMsg{ch: ch, cwd: cwd}
	}
}

func findNextLocalFile(m model) tea.Cmd {
	return func() tea.Msg {
		res, ok := <-m.localFileFinder

		if ok {
			// Okay now find the next one
			return foundLocalFileMsg(res)
		}
		// We're done
		if debug {
			log.Println("local file search finished")
		}
		return localFileSearchFinished{}
	}
}

func loadStash(m stashModel) tea.Cmd {
	return func() tea.Msg {
		if m.common.cc == nil {
			err := errors.New("no charm client")
			if debug {
				log.Println("error loading stash:", err)
			}
			return stashLoadErrMsg{err}
		}
		stash, err := m.common.cc.GetStash(m.serverPage)
		if err != nil {
			if debug {
				if _, ok := err.(charm.ErrAuthFailed); ok {
					log.Println("auth failure while loading stash:", err)
				} else {
					log.Println("error loading stash:", err)
				}
			}
			return stashLoadErrMsg{err}
		}
		if debug {
			log.Println("loaded stash page", m.serverPage)
		}
		return gotStashMsg(stash)
	}
}

func waitForStatusMessageTimeout(appCtx applicationContext, t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return statusMessageTimeoutMsg(appCtx)
	}
}

// ETC

// Convert a Gitcha result to an internal representation of a markdown
// document. Note that we could be doing things like checking if the file is
// a directory, but we trust that gitcha has already done that.
func localFileToMarkdown(cwd string, res gitcha.SearchResult) *markdown {
	md := &markdown{
		docType:   LocalDoc,
		localPath: res.Path,
		Markdown: charm.Markdown{
			Note:      stripAbsolutePath(res.Path, cwd),
			CreatedAt: res.Info.ModTime(),
		},
	}

	return md
}

func stripAbsolutePath(fullPath, cwd string) string {
	return strings.Replace(fullPath, cwd+string(os.PathSeparator), "", -1)
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
