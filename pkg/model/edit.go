package model

import (
	"os"
	"os/exec"

	"github.com/byxorna/jot/pkg/types/v1"
	tea "github.com/charmbracelet/bubbletea"
)

type reconcileEntryMsg v1.ID

func reconcileEntryCmd(id v1.ID) tea.Cmd {
	return func() tea.Msg { return reconcileEntryMsg(id) }
}

func (m *Model) EditCurrentEntry() tea.Cmd {
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
	// TODO: this somehow needs to tell the terminal to restore after blanking
	return tea.Sequentially(
		reconcileEntryCmd(md.ID),
		func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} },
	)
}
