// Source: https://github.com/maaslalani/slides/blob/main/internal/model/model.go
package model

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/byxorna/jot/pkg/db"
	"github.com/byxorna/jot/pkg/db/fs"
	"github.com/byxorna/jot/pkg/types/v1"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	//"github.com/maaslalani/slides/styles"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	delimiter    = "\n---\n"
	altDelimiter = "\n~~~\n"
)

type Model struct {
	Author   string
	Timeline []time.Time
	Date     time.Time

	Config v1.Config
	db.DB
	entry *v1.Entry

	Theme    glamour.TermRendererOption
	viewport viewport.Model
}

type fileWatchMsg struct{}

var fileInfo os.FileInfo

func (m Model) Init() tea.Cmd {
	if m.FileName == "" {
		return nil
	}
	fileInfo, _ = os.Stat(m.FileName)
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m *Model) initBackend() error {
	// TODO: switch here on backend type and load appropriate db provider
	loader, err := fs.New(m.Config.Directory)
	if err != nil {
		return err
	}
	m.DB = loader

	return fmt.Errorf("implement load() model")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ", "down", "k", "right", "l", "enter", "n":
			if m.Page < len(m.Slides)-1 {
				m.Page++
			}
		case "up", "j", "left", "h", "p":
			if m.Page > 0 {
				m.Page--
			}
		}

	case fileWatchMsg:
		newFileInfo, err := os.Stat(m.FileName)
		if err == nil && newFileInfo.ModTime() != fileInfo.ModTime() {
			fileInfo = newFileInfo
			_ = m.Load()
			if m.Page >= len(m.Slides) {
				m.Page = len(m.Slides) - 1
			}
		}
		return m, fileWatchCmd()
	}
	return m, nil
}

func (m Model) View() string {
	r, _ := glamour.NewTermRenderer(m.Theme, glamour.WithWordWrap(0))
	slide, err := r.Render(m.Content)
	if err != nil {
		slide = fmt.Sprintf("Error: Could not render markdown! (%v)", err)
	}
	// TODO: style output
	return slide
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
