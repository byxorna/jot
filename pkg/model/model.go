// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
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

type fileWatchMsg struct{}
type timeTickMsg struct{}

func (m Model) Init() tea.Cmd {
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	// TODO: improve this to not be so busy
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeTickMsg:
		m.Date = time.Now()
		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ", "down", "k", "right", "l", "enter", "n":
			// go to older entry
			if n, err := m.DB.Previous(m.Entry); err != nil {
				m.Entry = n
			} else {
				m.Err = err
			}
		case "up", "j", "left", "h", "p":
			// TODO(gabe): go to more recent entry
			if n, err := m.DB.Next(m.Entry); err != nil {
				m.Entry = n
			} else {
				m.Err = err
			}
		}

	case fileWatchMsg:
		// TODO: reload when changed?
		return m, fileWatchCmd()
	}
	return m, nil
}

func (m Model) View() string {
	r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithEmoji(), glamour.WithEnvironmentConfig(), glamour.WithWordWrap(0))
	if m.Entry == nil {
		return "no entry loaded"
	}
	md, err := r.Render(m.Entry.Content)
	if err != nil {
		m.Err = err

		return fmt.Sprintf("error rendering: %s", err.Error())
	}
	return md
	//slide = styles.Slide.Render(slide)

	//left := styles.Author.Render(m.Author) + styles.Date.Render(m.Date)
	//right := styles.Page.Render(fmt.Sprintf("%v", m))
	//status := styles.Status.Render(styles.JoinHorizontal(left, right, m.viewport.Width))
	//return styles.JoinVertical(slide, status, m.viewport.Height)
}

func readFile(path string) (string, error) {
	s, err := os.Stat(path)
	if err != nil {
		return "", errors.New("could not read file")
	}
	if s.IsDir() {
		return "", errors.New("can not read directory")
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		return "", errors.New("no slides provided")
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			return "", err
		}
	}

	return b.String(), nil
}
