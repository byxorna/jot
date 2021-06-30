package model

import (
	"fmt"
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
	oldW, oldH := m.common.width, m.common.height

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
	if err != nil {
		return m.stash.newStatusMessage(statusMessage{
			status:  errorStatusMessage,
			message: fmt.Sprintf("Error editing %s: %s", filename, err.Error()),
		})
	}

	var cmds []tea.Cmd
	cmds = append(cmds, reconcileEntryCmd(md.ID), func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} })
	if m.UseAltScreen {
		cmds = append(cmds, tea.EnterAltScreen)
	}
	return tea.Batch(cmds...)
}
