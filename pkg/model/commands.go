package model

import (
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type reloadEntryMsg struct{}
type fileWatchMsg struct{}

func (m Model) Init() tea.Cmd {
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	// TODO: improve this to not be so busy
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m Model) EditCurrentEntry() tea.Cmd {
	m.Mode = EditMode
	oldW, oldH := m.viewport.Width, m.viewport.Height

	filename := m.DB.StoragePath(m.Entry.ID)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	m.Err = cmd.Run()

	m.Mode = NormalMode
	return tea.Sequentially(
		reloadEntryCmd(),
		func() tea.Msg { return tea.WindowSizeMsg{Height: oldH, Width: oldW} })
}

func reloadEntryCmd() tea.Cmd {
	return func() tea.Msg { return reloadEntryMsg{} }
}

func errCmd(err error) tea.Cmd {
	return func() tea.Msg { return err }
}
