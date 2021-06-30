package model

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type updateViewMsg struct{}
type reconcileCurrentEntryMsg struct{}

func (m *Model) EditCurrentEntry() tea.Cmd {
	m.Mode = EditMode
	oldW, oldH := m.viewport.Width, m.viewport.Height

	md := m.stash.CurrentMarkdown()
	filename := m.DB.StoragePath(md.ID)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	m.handleError("edited entry", err)

	// TODO: reload entry manually here, because I dont know how to pipeline commands
	// in a way that will reload the entry, then repaint the screen :thinking:
	//e, err := m.Reconcile(m.EntryID)
	//m.handleError("reloaded entry", err)
	//m.EntryID = e.ID
	//m.Mode = ViewMode
	//m.viewport.YPosition = 0
	return tea.Batch(
		reconcileCurrentEntryCmd(),
		func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} },
	)
}

func updateViewCmd() tea.Cmd {
	return func() tea.Msg { return updateViewMsg{} }
}

func reconcileCurrentEntryCmd() tea.Cmd {
	return func() tea.Msg { return reconcileCurrentEntryMsg{} }
}

func errCmd(err error) tea.Cmd {
	return func() tea.Msg { return err }
}
