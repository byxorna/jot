package model

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) EditMarkdown(md *stashItem, previous *stashItem) tea.Cmd {
	oldW, oldH := m.common.width, m.common.height

	if md == nil {
		return func() tea.Msg { return errMsg{fmt.Errorf("no markdown id to edit: %s", md.Identifier())} }
	}

	filename := md.DocBackend.StoragePathDoc(md.Doc.Identifier())
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// +[num] position cursor on line num
	// +/{pat} position cursor on line num
	args := []string{"+/Notes"}
	if previous != nil {
		// we should show the previous day as the old diff
		prevFilename := md.DocBackend.StoragePathDoc(previous.Doc.Identifier())
		args = append(args, "-o2", filename, prevFilename)
	} else {
		args = append(args, filename)
	}
	cmd := exec.Command(editor, args...)

	{
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return m.stashModel.newStatusMessage(statusMessage{
				status:  errorStatusMessage,
				message: fmt.Sprintf("Error editing %s: %s", filename, err.Error()),
			})
		}

		go func() {
			defer stdinPipe.Close()
			io.Copy(stdinPipe, os.Stdin)
		}()
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return m.stashModel.newStatusMessage(statusMessage{
			status:  errorStatusMessage,
			message: fmt.Sprintf("Error editing %s: %s", filename, err.Error()),
		})
	}
	err := cmd.Wait()
	if err != nil {
		return m.stashModel.newStatusMessage(statusMessage{
			status:  errorStatusMessage,
			message: fmt.Sprintf("Error waiting for editor: %s", err.Error()),
		})
	}

	var cmds []tea.Cmd
	cmds = append(cmds, doReconcileStashItemCmd(md), func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} })
	if m.UseAltScreen {
		cmds = append(cmds, tea.EnterAltScreen)
	}
	return tea.Batch(cmds...)
}
