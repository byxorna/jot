// Source: https://raw.githubusercontent.com/charmbracelet/glow/d0737b41af48960a341e24327d9d5acb5b7d92aa/ui/stash.go
package model

import (
	"context"
	"fmt"
	"log"
	"os/user"
	"sort"
	"strings"
	"time"

	"github.com/byxorna/jot/pkg/config"
	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/plugins/calendar"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/byxorna/jot/pkg/version"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/sahilm/fuzzy"
	//te "github.com/muesli/termenv"
)

func newStashModel(common *commonModel, cfg *config.Config) (*stashModel, error) {
	sp := spinner.NewModel()
	sp.Spinner = spinner.Line
	sp.Style = lipgloss.NewStyle().Foreground(fuschia)
	sp.HideFor = time.Millisecond * 100
	sp.MinimumLifetime = time.Millisecond * 180
	sp.Start()

	ni := textinput.NewModel()
	ni.Prompt = stashTextInputPromptStyle("Memo: ")
	ni.CursorStyle = lipgloss.NewStyle().Foreground(fuschia)
	ni.CharLimit = noteCharacterLimit
	ni.Focus()

	si := textinput.NewModel()
	si.Prompt = stashTextInputPromptStyle("Find: ")
	si.CursorStyle = lipgloss.NewStyle().Foreground(fuschia)
	si.CharLimit = noteCharacterLimit
	si.Focus()

	// TODO: switch here on backend type and load appropriate db provider
	noteBackend, err := fs.New(cfg.Directory, true)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage provider: %w", err)
	}
	fmt.Printf("loaded %d entries\n", noteBackend.Count())

	var s []*section
	for _, sec := range cfg.Sections {
		switch sec.Plugin {
		case config.PluginTypeNotes:
			notes := newSectionModel(sec.Name, noteBackend, sec.Settings)
			s = append(s, &notes)
		case config.PluginTypeCalendar:
			cp, err := calendar.New(context.TODO())
			if err != nil {
				return nil, fmt.Errorf("%s failed to initialize: %w", sec.Plugin, err)
			}
			today := newSectionModel(sec.Name, cp, sec.Settings)
			s = append(s, &today)
		default:
			// TODO: maybe skip initialization? :thinking:
			return nil, fmt.Errorf("unsupported plugin %v for section name %s", sec.Plugin, sec.Name)
		}
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	m := stashModel{
		User:        *u,
		DB:          noteBackend,
		common:      common,
		config:      cfg,
		spinner:     sp,
		noteInput:   ni,
		filterInput: si,
		serverPage:  1,
		sections:    s,
	}

	return &m, nil
}

func newStashPaginator() paginator.Model {
	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.ActiveDot = brightGrayFg("•")
	p.InactiveDot = darkGrayFg("•")
	return p
}

const (
	stashIndent                = 1
	stashViewItemHeight        = 3 // height of stash note, including gap
	stashViewTopPadding        = 5 // logo, status bar, gaps
	stashViewBottomPadding     = 3 // pagination and gaps, but not help
	stashViewHorizontalPadding = 6
)

var (
	stashedStatusMessage        = statusMessage{normalStatusMessage, "Stashed!"}
	alreadyStashedStatusMessage = statusMessage{subtleStatusMessage, "Already stashed"}
	filterSectionID             = "filter"
)

var (
	stashTextInputPromptStyle styleFunc = newFgStyle(lib.YellowGreen)
	dividerDot                string    = darkGrayFg(" • ")
	dividerBar                string    = darkGrayFg(" │ ")
	offlineHeaderNote         string    = darkGrayFg("(Offline)")
)

type deletedStashedItemMsg int
type filteredStashItemMsg []*stashItem

// StashViewState is the high-level state of the file listing.
type StashViewState int

const (
	stashStateReady StashViewState = iota
	stashStateLoadingDocument
	stashStateShowingError
)

// filterState is the current filtering state in the file listing.
type filterState int

const (
	unfiltered    filterState = iota // no filter set
	filtering                        // user is actively setting a filter
	filterApplied                    // a filter is applied and user is not editing filter
)

// selectionState is the state of the currently selected document.
type selectionState int

const (
	selectionIdle = iota
	selectionSettingNote
	selectionPromptingDelete
)

// statusMessageType adds some context to the status message being sent.
type statusMessageType int

// Types of status messages.
const (
	normalStatusMessage statusMessageType = iota
	subtleStatusMessage
	errorStatusMessage
)

// statusMessage is an ephemeral note displayed in the UI.
type statusMessage struct {
	status  statusMessageType
	message string
}

// String returns a styled version of the status message appropriate for the
// given context.
func (s statusMessage) String() string {
	switch s.status {
	case subtleStatusMessage:
		return dimGreenFg(s.message)
	case errorStatusMessage:
		return redFg(s.message)
	default:
		return greenFg(s.message)
	}
}

type stashModel struct {
	db.DB

	User               user.User
	common             *commonModel
	config             *config.Config
	err                error
	spinner            spinner.Model
	noteInput          textinput.Model
	filterInput        textinput.Model
	viewState          StashViewState
	filterState        filterState
	selectionState     selectionState
	showFullHelp       bool
	showStatusMessage  bool
	statusMessage      statusMessage
	statusMessageTimer *time.Timer

	// Available document sections we can cycle through. We use a slice, rather
	// than a map, because order is important.
	sections []*section

	// Index of the section we're currently looking at
	sectionIndex int

	// The master set of markdown documents we're working with.
	markdowns []*stashItem

	// Markdown documents we're currently displaying. Filtering, toggles and so
	// on will alter this slice so we can show what is relevant. For that
	// reason, this field should be considered ephemeral.
	filteredStashItems []*stashItem

	// Page we're fetching stash items from on the server, which is different
	// from the local pagination. Generally, the server will return more items
	// than we can display at a time so we can paginate locally without having
	// to fetch every time.
	serverPage int
}

func (m *stashModel) hasSection(identifier string) bool {
	for _, v := range m.sections {
		if identifier == v.Identifier() {
			return true
		}
	}
	return false
}

func (m *stashModel) paginator() *paginator.Model {
	return &m.focusedSection().paginator
}

func (m *stashModel) setPaginator(p paginator.Model) {
	m.focusedSection().paginator = p
}

func (m *stashModel) cursor() int {
	return m.focusedSection().cursor
}

func (m *stashModel) setCursor(i int) {
	m.focusedSection().cursor = i
}

func (m *stashModel) setSize(width, height int) {
	m.common.width = width
	m.common.height = height

	m.noteInput.Width = width - stashViewHorizontalPadding*2 - ansi.PrintableRuneWidth(m.noteInput.Prompt)
	m.filterInput.Width = width - stashViewHorizontalPadding*2 - ansi.PrintableRuneWidth(m.filterInput.Prompt)

	m.updatePagination()
}

func (m *stashModel) resetFiltering() {
	m.filterState = unfiltered
	m.filterInput.Reset()
	m.filteredStashItems = nil

	sort.Stable(markdownsByLocalFirst(m.markdowns))

	// If the filtered section is present (it's always at the end) slice it out
	// of the sections slice to remove it from the UI.
	if m.sections[len(m.sections)-1].Identifier() == filterSectionID {
		m.sections = m.sections[:len(m.sections)-1]
	}

	// If the current section is out of bounds (it would be if we cut down the
	// slice above) then return to the first section.
	if m.sectionIndex > len(m.sections)-1 {
		m.sectionIndex = 0
	}

	// Update pagination after we've switched sections.
	m.updatePagination()
}

// Is a filter currently being applied?
func (m *stashModel) filterApplied() bool {
	return m.filterState != unfiltered
}

// Should we be updating the filter?
func (m *stashModel) shouldUpdateFilter() bool {
	// If we're in the middle of setting a note don't update the filter so that
	// the focus won't jump around.
	return m.filterApplied() //&& m.selectionState != selectionSettingNote
}

// Update pagination according to the amount of markdowns for the current
// state.
func (m *stashModel) updatePagination() {
	_, helpHeight := m.helpView()

	availableHeight := m.common.height -
		stashViewTopPadding -
		helpHeight -
		stashViewBottomPadding

	m.paginator().PerPage = max(1, availableHeight/stashViewItemHeight)

	if pages := len(m.getVisibleStashItems()); pages < 1 {
		m.paginator().SetTotalPages(1)
	} else {
		m.paginator().SetTotalPages(pages)
	}

	// Make sure the page stays in bounds
	if m.paginator().Page >= m.paginator().TotalPages-1 {
		m.paginator().Page = max(0, m.paginator().TotalPages-1)
	}
}

// MarkdownIndex returns the index of the currently selected markdown item.
func (m *stashModel) markdownIndex() int {
	return m.paginator().Page*m.paginator().PerPage + m.cursor()
}

// Return the current selected markdown in the stash.
func (m *stashModel) CurrentStashItem() (*stashItem, error) {
	i := m.markdownIndex()

	mds := m.getVisibleStashItems()
	if i < 0 || len(mds) == 0 || len(mds) <= i {
		return nil, fmt.Errorf("no item focused")
	}

	return mds[i], nil
}

func (m *stashModel) hasMarkdown(md *stashItem) bool {
	for _, existing := range m.markdowns {
		if md.Identifier() == existing.Identifier() {
			return true
		}
	}
	return false
}

// Adds markdown documents to the model.
func (m *stashModel) addMarkdowns(mds ...*stashItem) {
	for _, md := range mds {
		if m.hasMarkdown(md) {
			// replace existing note
			mds, err := deleteMarkdown(m.markdowns, md)
			if err == nil {
				m.markdowns = mds
			}
		}
		m.markdowns = append(m.markdowns, md)
	}

	if !m.filterApplied() {
		sort.Stable(markdownsByLocalFirst(m.markdowns))
	}
	m.updatePagination()
}

func (m *stashModel) getVisibleStashItems() []*stashItem {
	if m.filterState == filtering || m.focusedSection().Identifier() == filterSectionID {
		return m.filteredStashItems
	}

	backend := m.focusedSection().DocBackend
	l, _ := backend.List()
	items := make([]*stashItem, len(l))
	for i, d := range l {
		items[i] = AsStashItem(d, backend)
	}

	return items
}

// Command for opening a markdown document in the pager. Note that this also
// alters the model.
func (m *stashModel) viewCurrentNoteCmd() tea.Cmd {
	m.viewState = stashStateLoadingDocument

	//return tea.Batch(loadLocalMarkdown(md), spinner.Tick)
	return tea.Batch(spinner.Tick)
}

func (m *stashModel) newStatusMessage(sm statusMessage) tea.Cmd {
	m.showStatusMessage = true
	m.statusMessage = sm
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(statusMessageTimeout)
	return waitForStatusMessageTimeout(stashContext, m.statusMessageTimer)
}

func (m *stashModel) hideStatusMessage() {
	m.showStatusMessage = false
	m.statusMessage = statusMessage{}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
}

func (m *stashModel) moveCursorUp() {
	m.setCursor(m.cursor() - 1)
	if m.cursor() < 0 && m.paginator().Page == 0 {
		// Stop
		m.setCursor(0)
		return
	}

	if m.cursor() >= 0 {
		return
	}
	// Go to previous page
	m.paginator().PrevPage()

	m.setCursor(m.paginator().ItemsOnPage(len(m.getVisibleStashItems())) - 1)
}

func (m *stashModel) moveCursorDown() {
	itemsOnPage := m.paginator().ItemsOnPage(len(m.getVisibleStashItems()))

	m.setCursor(m.cursor() + 1)
	if m.cursor() < itemsOnPage {
		return
	}

	if !m.paginator().OnLastPage() {
		m.paginator().NextPage()
		m.setCursor(0)
		return
	}

	// During filtering the cursor position can exceed the number of
	// itemsOnPage. It's more intuitive to start the cursor at the
	// topmost position when moving it down in this scenario.
	if m.cursor() > itemsOnPage {
		m.setCursor(0)
		return
	}
	m.setCursor(itemsOnPage - 1)
}

// INIT

// UPDATE

func (m *stashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := m.update(msg)
	return tea.Model(newModel), cmd
}

func (m *stashModel) update(msg tea.Msg) (*stashModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case errMsg:
		m.err = msg

	case stashItemUpdateMsg:
		updatedItem := msg
		m.addMarkdowns(updatedItem)
		return m, nil

	case filteredStashItemMsg:
		m.filteredStashItems = msg
		return m, nil

	case spinner.TickMsg:
		// TODO: for now, just stub this out. Need to figure out what to do
		// when we have multiple doc types that may load asynchronously
		loading := false

		openingDocument := m.viewState == stashStateLoadingDocument
		spinnerVisible := m.spinner.Visible()

		if loading || openingDocument || spinnerVisible {
			newSpinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = newSpinnerModel
			cmds = append(cmds, cmd)
		}

	// Note: mechanical stuff related to stash failure is handled in the parent
	// update function.
	case stashFailMsg:
		m.err = msg.err
		cmds = append(cmds, m.newStatusMessage(statusMessage{
			status:  errorStatusMessage,
			message: fmt.Sprintf("Couldn’t stash ‘%s’", msg.markdown.Identifier()),
		}))

	case statusMessageTimeoutMsg:
		if applicationContext(msg) == stashContext {
			m.hideStatusMessage()
		}
	}

	if m.filterState == filtering {
		cmds = append(cmds, m.handleFiltering(msg))
		return m, tea.Batch(cmds...)
	}

	// Updates per the current state
	switch m.viewState {
	case stashStateReady:
		cmds = append(cmds, m.handleDocumentBrowsing(msg))
	case stashStateShowingError:
		// Any key exists the error view
		if _, ok := msg.(tea.KeyMsg); ok {
			m.viewState = stashStateReady
		}
	}

	return m, tea.Batch(cmds...)
}

// Updates for when a user is browsing the markdown listing.
func (m *stashModel) handleDocumentBrowsing(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	numDocs := len(m.getVisibleStashItems())

	switch msg := msg.(type) {
	// Handle keys
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "ctrl+k", "up":
			m.moveCursorUp()

		case "j", "ctrl+j", "down":
			m.moveCursorDown()

		// Go to the very start
		case "home", "g":
			m.paginator().Page = 0
			m.setCursor(0)

		// Go to the very end
		case "end", "G":
			m.paginator().Page = m.paginator().TotalPages - 1
			m.setCursor(m.paginator().ItemsOnPage(numDocs) - 1)

		// Clear filter (if applicable)
		case "esc":
			if m.filterApplied() {
				m.resetFiltering()
			} else if m.viewState == stashStateLoadingDocument {
				// if escape pressed when we have loaded a document, go back to ready view
				m.viewState = stashStateReady
			}

		// Next section
		case "tab", "L":
			if len(m.sections) == 0 || m.filterState == filtering {
				break
			}
			m.sectionIndex++
			if m.sectionIndex >= len(m.sections) {
				m.sectionIndex = 0
			}
			m.updatePagination()

		// Previous section
		case "shift+tab", "H":
			if len(m.sections) == 0 || m.filterState == filtering {
				break
			}
			m.sectionIndex--
			if m.sectionIndex < 0 {
				m.sectionIndex = len(m.sections) - 1
			}
			m.updatePagination()

		// Open document
		case "enter", "v":
			m.hideStatusMessage()

			if numDocs == 0 {
				break
			}

			cmds = append(cmds, m.viewCurrentNoteCmd())

		// Filter your notes
		case "/":
			m.hideStatusMessage()

			// Build values we'll filter against
			for _, md := range m.markdowns {
				md.buildFilterValue()
			}

			m.filteredStashItems = m.getVisibleStashItems()

			m.paginator().Page = 0
			m.setCursor(0)
			m.filterState = filtering
			m.filterInput.CursorEnd()
			m.filterInput.Focus()
			return textinput.Blink

		// Set note
		//case "m":
		//	m.hideStatusMessage()

		//	if numDocs == 0 {
		//		break
		//	}

		//	md := m.CurrentStashItem()
		//	isUserMarkdown := md.docType == StashedDoc || md.docType == ConvertedDoc
		//	isSettingNote := m.selectionState == selectionSettingNote
		//	isPromptingDelete := m.selectionState == selectionPromptingDelete

		//	if isUserMarkdown && !isSettingNote && !isPromptingDelete {
		//		m.selectionState = selectionSettingNote
		//		m.noteInput.SetValue("")
		//		m.noteInput.CursorEnd()
		//		return textinput.Blink
		//	}

		// Prompt for deletion
		//case "x":
		//	m.hideStatusMessage()

		//	validState := m.viewState == stashStateReady &&
		//		m.selectionState == selectionIdle

		//	if numDocs == 0 && !validState {
		//		break
		//	}

		//	md := m.CurrentStashItem()
		//	if md == nil {
		//		break
		//	}

		//	t := md.docType
		//	if t == StashedDoc || t == ConvertedDoc {
		//		m.selectionState = selectionPromptingDelete
		//	}

		// Toggle full help
		case "?":
			m.showFullHelp = !m.showFullHelp
			m.updatePagination()

		// Show errors
		case "!":
			if m.err != nil && m.viewState == stashStateReady {
				m.viewState = stashStateShowingError
				return nil
			}
		}
	}

	// Update paginator. Pagination key handling is done here, but it could
	// also be moved up to this level, in which case we'd use model methods
	// like model.PageUp().
	newPaginatorModel, cmd := m.paginator().Update(msg)
	m.setPaginator(newPaginatorModel)
	cmds = append(cmds, cmd)

	// Extra paginator keystrokes
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "b", "u":
			m.paginator().PrevPage()
		case "f", "d":
			m.paginator().NextPage()
		}
	}

	// Keep the index in bounds when paginating
	itemsOnPage := m.paginator().ItemsOnPage(len(m.getVisibleStashItems()))
	if m.cursor() > itemsOnPage-1 {
		m.setCursor(max(0, itemsOnPage-1))
	}

	return tea.Batch(cmds...)
}

// Updates for when a user is being prompted whether or not to delete a
// markdown item.

// Updates for when a user is in the filter editing interface.
func (m *stashModel) handleFiltering(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	{ // if there is no filter section, add one immediately at the end
		if m.sections[len(m.sections)-1].Identifier() != filterSectionID {
			filterSection := newSectionModel(filterSectionID, m.DB, map[string]string{})
			m.sections = append(m.sections, &filterSection)
		}
		m.sectionIndex = len(m.sections) - 1
	}

	// Handle keys
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			// Cancel filtering
			m.resetFiltering()
		case "enter", "tab", "shift+tab", "ctrl+k", "up", "ctrl+j", "down":
			m.hideStatusMessage()

			if len(m.markdowns) == 0 {
				break
			}

			h := m.getVisibleStashItems()

			// If we've filtered down to nothing, clear the filter
			if len(h) == 0 {
				m.viewState = stashStateReady
				m.resetFiltering()
				break
			}

			// When there's only one filtered markdown left we can just
			// "open" it directly
			if len(h) == 1 {
				m.viewState = stashStateReady
				m.resetFiltering()
				cmds = append(cmds, m.viewCurrentNoteCmd())
				break
			}

			// Add new section if it's not present
			// TODO: this should be moved elsewhere, because the way sections are inserted/removed is not ideal
			// WRT ordering
			/*
				if m.sections[len(m.sections)-1].Identifier() != filterSectionID {
					filterSection := newSectionModel(filterSectionID, m.DB, map[string]string{})
					m.sections = append(m.sections, &filterSection)
				}
				m.sectionIndex = len(m.sections) - 1
			*/

			m.filterInput.Blur()

			m.filterState = filterApplied
			if m.filterInput.Value() == "" {
				m.resetFiltering()
			}
		}
	}

	// Update the filter text input component
	newFilterInputModel, inputCmd := m.filterInput.Update(msg)
	currentFilterVal := m.filterInput.Value()
	newFilterVal := newFilterInputModel.Value()
	m.filterInput = newFilterInputModel
	cmds = append(cmds, inputCmd)

	// If the filtering input has changed, request updated filtering
	if newFilterVal != currentFilterVal {
		cmds = append(cmds, filterMarkdowns(*m))
	}

	// Update pagination
	m.updatePagination()

	return tea.Batch(cmds...)
}

// VIEW

func (m *stashModel) View() string {
	var s string
	switch m.viewState {
	case stashStateShowingError:
		return errorView(m.err, false)
	case stashStateLoadingDocument:
		s += " " + m.spinner.View() + " Loading document..."
	case stashStateReady:
		loadingIndicator := " "
		if m.focusedSection().Status() == v1.StatusSynchronizing || m.spinner.Visible() {
			loadingIndicator = m.spinner.View()
		}

		var header string
		//switch m.selectionState {
		//case selectionPromptingDelete:
		//	header = redFg("Delete this item from your stash? ") + faintRedFg("(y/N)")
		//case selectionSettingNote:
		//	header = yellowFg("Set the memo for this item?")
		//}

		// Only draw the normal header if we're not using the header area for
		// something else (like a note or delete prompt).
		if header == "" {
			header = m.headerView()
		}

		// Rules for the logo, filter and status message.
		logoOrFilter := " "
		if m.showStatusMessage && m.filterState == filtering {
			logoOrFilter += m.statusMessage.String()
		} else if m.filterState == filtering {
			logoOrFilter += m.filterInput.View()
		} else {
			logoOrFilter += glowLogoView(" Jot ", fmt.Sprintf(" version %s", version.Version))
			if m.showStatusMessage {
				logoOrFilter += "  " + m.statusMessage.String()
			}
		}
		logoOrFilter = truncate.StringWithTail(logoOrFilter, uint(m.common.width-1), ellipsis)

		help, helpHeight := m.helpView()

		populatedView := m.populatedView()
		populatedViewHeight := strings.Count(populatedView, "\n") + 2

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		availHeight := m.common.height -
			stashViewTopPadding -
			populatedViewHeight -
			helpHeight -
			stashViewBottomPadding
		blankLines := strings.Repeat("\n", max(0, availHeight))

		var pagination string
		if m.paginator().TotalPages > 1 {
			pagination = m.paginator().View()

			// If the dot pagination is wider than the width of the window
			// use the arabic paginator.
			if ansi.PrintableRuneWidth(pagination) > m.common.width-stashViewHorizontalPadding {
				// Copy the paginator since m.paginator() returns a pointer to
				// the active paginator and we don't want to mutate it. In
				// normal cases, where the paginator is not a pointer, we could
				// safely change the model parameters for rendering here as the
				// current model is discarded after reuturning from a View().
				// One could argue, in fact, that using pointers in
				// a functional framework is an antipattern and our use of
				// pointers in our model should be refactored away.
				var p paginator.Model = *(m.paginator())
				p.Type = paginator.Arabic
				pagination = lib.Subtle(p.View())
			}

		}

		s += fmt.Sprintf(
			"%s%s\n\n  %s\n\n%s\n\n%s  %s\n\n%s",
			loadingIndicator,
			logoOrFilter,
			header,
			populatedView,
			blankLines,
			pagination,
			help,
		)
	}
	return "\n" + indent(s, stashIndent)
}

func glowLogoView(text, additional string) string {
	return purpleStatusPillStyle.Render(text) + brightGrayFg(additional)
}

func (m *stashModel) headerView() string {
	var sections []string

	// Tabs
	for i, v := range m.sections {
		var s string
		if v.Identifier() == filterSectionID {
			s = fmt.Sprintf("%d %s “%s”", len(m.filteredStashItems), v.Identifier(), m.filterInput.Value())
		} else {
			s = v.TabTitle()
		}

		if m.sectionIndex == i {
			if m.IsFiltering() && v.Identifier() == filterSectionID {
				s = dullYellowFg(s)
			} else {
				s = selectedTabColor(s)
			}
		} else {
			s = tabColor(s)
		}

		sections = append(sections, s)
	}

	return strings.Join(sections, dividerBar)
}

func (m stashModel) populatedView() string {
	mds := m.getVisibleStashItems()

	var b strings.Builder

	// Empty states
	if len(mds) == 0 {
		f := func(s string) {
			b.WriteString("  " + grayFg(s))
		}

		thisFocusedSection := m.sections[m.sectionIndex]
		if thisFocusedSection.Identifier() == filterSectionID {
			return ""
		}
		switch thisFocusedSection.DocBackend.Status() {
		case v1.StatusUninitialized:
			f(fmt.Sprintf("Still initializing %vs...", thisFocusedSection.DocType()))
		default:
			f(fmt.Sprintf("No %vs found.", thisFocusedSection.DocType()))
		}
	} else {
		start, end := m.paginator().GetSliceBounds(len(mds))
		docs := mds[start:end]

		for i, doc := range docs {
			rendered := stashItemView(m.common.width, m.cursor() == i, m.filterState == filtering, m.filterInput.Value(), len(m.getVisibleStashItems()), doc)
			fmt.Fprint(&b, rendered)
			if i != len(docs)-1 {
				fmt.Fprintf(&b, "\n\n")
			}
		}
	}

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space where stash items
	// would have been.
	itemsOnPage := m.paginator().ItemsOnPage(len(mds))
	if itemsOnPage < m.paginator().PerPage {
		n := (m.paginator().PerPage - itemsOnPage) * stashViewItemHeight
		if len(mds) == 0 {
			n -= stashViewItemHeight - 1
		}
		for i := 0; i < n; i++ {
			fmt.Fprint(&b, "\n")
		}
	}

	return b.String()
}

// COMMANDS

func filterMarkdowns(m stashModel) tea.Cmd {
	return func() tea.Msg {
		if m.filterInput.Value() == "" || !m.filterApplied() {
			return filteredStashItemMsg(m.getVisibleStashItems()) // return everything
		}

		targets := []string{}
		mds := m.getVisibleStashItems()

		for _, t := range mds {
			targets = append(targets, t.filterValue)
		}

		ranks := fuzzy.Find(m.filterInput.Value(), targets)
		sort.Stable(ranks)

		filtered := []*stashItem{}
		for _, r := range ranks {
			filtered = append(filtered, mds[r.Index])
		}

		// TODO: figure out whether this totally clobbers the ranking that is performed earlier
		// because I would rather the entries stay in order when filtering, instead of sorting by
		// fuzzy finding
		sort.Stable(markdownsByLocalFirst(filtered))

		return filteredStashItemMsg(filtered)
	}
}

// Delete a markdown from a slice of markdowns.
func deleteMarkdown(markdowns []*stashItem, target *stashItem) ([]*stashItem, error) {
	index := -1

	// Operate on a copy to avoid any pointer weirdness
	mds := make([]*stashItem, len(markdowns))
	copy(mds, markdowns)

	for i, v := range mds {
		if v.Identifier() == target.Identifier() {
			index = i
			break
		}
	}

	if index == -1 {
		err := fmt.Errorf("could not find markdown to delete")
		if debug {
			log.Println(err)
		}
		return nil, err
	}

	return append(mds[:index], mds[index+1:]...), nil
}

func (m *stashModel) Init() tea.Cmd { return nil }

func (m *stashModel) FocusedSection() Section {
	return m.focusedSection()
}

func (m *stashModel) focusedSection() *section {
	// FIXME: this is pretty sus - our focused
	if m.sectionIndex > len(m.sections) {
		m.sectionIndex = 0
		return m.sections[m.sectionIndex]
	}
	return m.sections[m.sectionIndex]
}

func (m *stashModel) Sections() []Section {
	s := make([]Section, len(m.sections))
	for i, sx := range m.sections {
		s[i] = Section(sx)
	}
	return s
}

func (m *stashModel) IsFiltering() bool {
	return m.filterApplied()
}

func (m *stashModel) ReloadNoteCollectionCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.ListAll()
		if err != nil {
			return errMsg{err}
		}

		mds := make([]*stashItem, len(entries))
		for i, e := range entries {
			locale := e
			md := AsStashItem(locale, m.DB)
			mds[i] = md
		}

		return stashItemCollectionReconcileMsg(mds)
	}
}

// Open either the appropriate entry for today, or create a new one
func (m *stashModel) createDaysEntryCmd(day time.Time) (*stashModel, tea.Cmd) {
	return m, func() tea.Msg {
		if entries, err := m.DB.ListAll(); err == nil {
			// if the most recent entry isnt the same as our expected filename, create a new entry for today
			expectedFilename := day.Format(fs.StorageFilenameFormat)
			if len(entries) == 0 || (len(entries) > 0 && entries[0].Metadata.CreationTimestamp.Format(fs.StorageFilenameFormat) != expectedFilename) {
				_, err := m.DB.CreateOrUpdateNote(&v1.Note{
					Metadata: v1.NoteMetadata{
						Author: m.User.Username,
						Title:  TitleFromTime(day, m.config.StartWorkHours, m.config.EndWorkHours),
						Tags:   DefaultTagsForTime(day, m.config.HolidayTags, m.config.WorkdayTags, m.config.WeekendTags),
						Labels: map[string]string{},
					},
					Content: m.config.EntryTemplate,
				})
				if err != nil {
					return errMsg{fmt.Errorf("unable to create new entry: %w", err)}
				}
				// TODO: we should not need to reload the whole collection, but I dunno how to make this work otherwise
				return m.ReloadNoteCollectionCmd()
			} else {
				return m.newStatusMessage(statusMessage{
					status:  normalStatusMessage,
					message: fmt.Sprintf("Entry %s already exists", expectedFilename),
				})
			}
		} else {
			return errMsg{fmt.Errorf("unable to list entries: %w", err)}
		}
	}
}
