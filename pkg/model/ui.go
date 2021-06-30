// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/ui.go
package model

import (
	"fmt"
	"os"
	"strings"
	"time"

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

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type initLocalFileSearchMsg struct {
	cwd string
	ch  chan gitcha.SearchResult
}
type foundLocalFileMsg gitcha.SearchResult
type statusMessageTimeoutMsg applicationContext
type stashFailMsg struct {
	err      error
	markdown markdown
}
type entryCollectionLoadedMsg []*markdown
type entryUpdateMsg markdown

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

// unloadDocument unloads a document from the pager. Note that while this
// method alters the model we also need to send along any commands returned.
func (m *Model) unloadDocument() []tea.Cmd {
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

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, spinner.Tick, loadEntriesToStash(&m)) //, updateViewCmd())
	return tea.Batch(cmds...)
	//return loadEntriesToStash(&m)
}

// From orig
//
/*
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Date = time.Now()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.viewportReady {
			// Since this program is using the full size of the viewport we need
			// to wait until we've received the window dimensions before we
			// can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.Model{Width: msg.Width, Height: msg.Height}
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = UseHighPerformanceRendering
			m.viewport.SetContent(m.content)
			m.viewportReady = true
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			switch m.Mode {
			case ViewMode:
				m.LogUserNotice("all entries")
				m.Mode = ListMode
			case HelpMode:
				m.LogUserNotice("view mode")
				m.Mode = ViewMode
			}
			return m, nil
		case "v", "enter":
			switch m.Mode {
			case ListMode:
				// TODO: should we focus the appropriate entry ID?
				m.Mode = ViewMode
			}
			return m, nil
		case "?":
			m.LogUserNotice("use esc to return")
			m.Mode = HelpMode
			return m, nil
		case "r":
			m.LogUserNotice("reloading entry")
			return m, reloadEntryCmd()
		case "e":
			m.LogUserNotice("editing entry")
			cmd := m.EditCurrentEntry()
			return m, cmd
		case "h":
			e, err := m.DB.Next(m.EntryID)
			if err == db.ErrNoNextEntry {
				return m, nil
			}
			m.handleError("next entry", err)
			m.EntryID = e.ID
			return m, updateViewCmd()
		case "l":
			e, err := m.DB.Previous(m.EntryID)
			if err == db.ErrNoPrevEntry {
				return m, nil
			}
			m.handleError("previous entry", err)
			m.EntryID = e.ID
			return m, updateViewCmd()
		}

	case updateViewMsg:
		err := m.UpdateContent()
		if err != nil {
			m.LogUserError(err)
		}
		return m, nil
	case reloadEntryMsg:
		_, err := m.Reconcile(m.EntryID)
		if err != nil {
			m.LogUserError(err)
		}
		return m, updateViewCmd()
	}

	// Because we're using the viewport's default update function (with pager-
	// style navigation) it's important that the viewport's update function:
	//
	// * Receives messages from the Bubble Tea runtime
	// * Returns commands to the Bubble Tea runtime
	//
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}
*/

// Update handles messages emitted by the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Date = time.Now()

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

		case "e":
			switch m.state {
			case stateShowStash:
				if m.stash.filterState != filtering && m.pager.state == pagerStateBrowse {
					return m, m.EditCurrentEntry()
				}
			}
		case "r":
			if m.state == stateShowStash && m.stash.filterState != filtering && m.pager.state == pagerStateBrowse {
				cmds = append(cmds, m.stash.newStatusMessage(statusMessage{
					status:  subtleStatusMessage,
					message: fmt.Sprintf("Reloading %s", m.stash.CurrentMarkdown().LocalPath),
				}),
					reconcileCurrentEntryCmd(),
				)
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

	case fetchedMarkdownMsg:
		// We've loaded a markdown file's contents for rendering
		m.pager.currentDocument = *msg
		//msg.Content = string(utils.RemoveFrontmatter([]byte(msg.Content)))
		cmds = append(cmds, renderWithGlamour(m.pager, msg.Content))

	case contentRenderedMsg:
		m.state = stateShowDocument

	case entryCollectionLoadedMsg, entryUpdateMsg:
		stashModel, cmd := m.stash.update(msg)
		m.stash = stashModel
		cmds = append(cmds, cmd)

	case filteredMarkdownMsg:
		if m.state == stateShowDocument {
			newStashModel, cmd := m.stash.update(msg)
			m.stash = newStashModel
			cmds = append(cmds, cmd)
		}

	case reconcileCurrentEntryMsg:
		md := m.stash.CurrentMarkdown()
		reconciled, err := m.Reconcile(md.ID)
		if err != nil {
			m.LogUserError(err)
		}
		path := m.StoragePath(reconciled.ID)
		cmds = append(cmds,
			m.stash.newStatusMessage(statusMessage{
				status:  subtleStatusMessage,
				message: fmt.Sprintf("!!RECONCILIATION %s", m.stash.CurrentMarkdown().LocalPath),
			}),
			func() tea.Msg { return entryUpdateMsg(AsMarkdown(path, *reconciled)) })
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

func (m Model) View() string {
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

func findLocalFiles(m *Model) tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

//func findNextLocalFile(m model) tea.Cmd {
//	return func() tea.Msg {
//		res, ok := <-m.localFileFinder
//
//		if ok {
//			// Okay now find the next one
//			return foundLocalFileMsg(res)
//		}
//		// We're done
//		if debug {
//			log.Println("local file search finished")
//		}
//		return localFileSearchFinished{}
//	}
//}

func loadEntriesToStash(m *Model) tea.Cmd {
	return func() tea.Msg {
		entries, err := m.ListAll()
		if err != nil {
			return errMsg{err}
		}

		mds := make([]*markdown, len(entries))
		for i, e := range entries {
			md := AsMarkdown(m.DB.StoragePath(e.ID), *e)
			mds[i] = &md
		}

		return entryCollectionLoadedMsg(mds)
	}
}

func waitForStatusMessageTimeout(appCtx applicationContext, t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return statusMessageTimeoutMsg(appCtx)
	}
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
