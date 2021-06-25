// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	db.DB

	Author   string
	Timeline []time.Time
	Date     time.Time
	Config   v1.Config
	Entry    *v1.Entry
	Err      error

	viewport viewport.Model
}

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
	filename := m.DB.StoragePath(m.Entry)
	fmt.Printf("editing %s", filename)
	editor := "nvim"

	//	{
	// plumb terminal in to a pipe so we can reconnect things after
	// invoking the editor
	//r, w, err := os.Pipe()
	//if err != nil {
	//	return errCmd(err)
	//}

	//
	//		syscall.Dup2(int(w.Fd()), syscall.Stdout)
	//		log("Writing to stdout (actually to pipe)")
	//
	//		fmt.Print("Hello, world!")
	//
	//		log("Closing write-end of pipe")
	//		w.Close()
	//
	//		if closeStdout {
	//			log("Closing stdout")
	//			syscall.Close(syscall.Stdout)
	//		}
	//
	//		var b bytes.Buffer
	//
	//		log("Copying from pipe to buffer")
	//		io.Copy(&b, r)
	//
	//		log("Restoring stdout")
	//		syscall.Dup2(oldStdout, syscall.Stdout)
	//
	//		return b.String()
	//	}
	//

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	m.Err = cmd.Run()
	return reloadEntryCmd()
}

func reloadEntryCmd() tea.Cmd {
	return func() tea.Msg { return reloadEntryMsg{} }
}
func errCmd(err error) tea.Cmd {
	fmt.Printf("error: %s", err.Error())
	return func() tea.Msg { return err }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Date = time.Now()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "e":
			return m, m.EditCurrentEntry()
		case " ", "down", "k", "right", "l", "enter", "n":
			// go to older entry
			if n, err := m.DB.Previous(m.Entry); err != nil {
				fmt.Printf("got err: %v\n", err.Error())
				m.Err = err
			} else {
				m.Entry = n
			}
			return m, nil
		case "up", "j", "left", "h", "p":
			// TODO(gabe): go to more recent entry
			if n, err := m.DB.Next(m.Entry); err != nil {
				fmt.Printf("got err: %v\n", err.Error())
				m.Err = err
			} else {
				m.Entry = n
			}
			return m, nil
		}

	case reloadEntryMsg:
		m.Entry, m.Err = m.DB.Get(m.Entry.ID, true)
		fmt.Printf("reloaded %d: %v\n", m.Entry.ID, m.Err)
		return m, nil
	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}
