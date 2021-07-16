package model

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/byxorna/jot/pkg/db"
	tea "github.com/charmbracelet/bubbletea"
)

type reconcileStashItemMsg struct {
	item    *stashItem
	backend db.DocBackend
}

func reconcileStashItemCmd(md *stashItem, backend db.DocBackend) tea.Cmd {
	return func() tea.Msg { return reconcileStashItemMsg{md, backend} }
}

func (m *Model) EditMarkdown(md *stashItem) tea.Cmd {
	oldW, oldH := m.common.width, m.common.height

	if md == nil {
		return func() tea.Msg { return errMsg{fmt.Errorf("no markdown id to edit: %s", md.Identifier())} }
	}

	backend := m.DB // TODO FIXME for now hardcode this backend to point to our known local files backend

	filename := path.Join(backend.StoragePath(), md.Doc.Identifier())
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, filename)

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
	cmds = append(cmds, reconcileStashItemCmd(md, backend), func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} })
	if m.UseAltScreen {
		cmds = append(cmds, tea.EnterAltScreen)
	}
	return tea.Batch(cmds...)
}
