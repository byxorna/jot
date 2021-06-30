package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/charm/ui/common"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
	"github.com/enescakir/emoji"
	"github.com/lucasb-eyer/go-colorful"
	te "github.com/muesli/termenv"
)

type action struct {
	key   string
	short string
}

const (
	columnWidth = 30
)

var (
	controls = []action{
		{key: "?", short: "help"},
		{key: "esc", short: "back"},
		{key: "e", short: "edit"},
		{key: "/", short: "search"},
		{key: "j", short: "previous day"},
		{key: "k", short: "next day"},
		{key: "q", short: "quit"},
	}

	// Style definitions.

	// General.

	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	focus     = lipgloss.AdaptiveColor{Light: "#111111", Dark: "#E7E7E7"}
	// TODO change this color, its used only in history for completed days
	dim      = lipgloss.AdaptiveColor{Light: "#0000ff", Dark: "#000099"}
	darkGray = lipgloss.AdaptiveColor{Light: "#0000ff", Dark: "#333333"}

	fuschia   = lipgloss.Color("205")
	orangeRed = lipgloss.Color("202")
	red       = lipgloss.Color("197")
	gray      = lipgloss.Color("8")
	teal      = lipgloss.Color("6")
	purple    = lipgloss.Color("5")

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()

	url = lipgloss.NewStyle().Foreground(special).Render

	// Tabs.

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	activeTab = tab.Copy().Border(activeTabBorder, true)

	tabGap = tab.Copy().
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)

	// Title.

	titleStyle = lipgloss.NewStyle().
			MarginLeft(1).
			MarginRight(5).
			Padding(0, 1).
			Italic(true).
			Foreground(lipgloss.Color("#FFF7DB")).
			SetString("Lip Gloss")

	descStyle = lipgloss.NewStyle().MarginTop(1)

	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle)

	// Dialog.

	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)

	// List.

	list = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, false).
		BorderForeground(subtle).
		MarginRight(2).
		Height(8).
		Width(columnWidth + 1)

	listHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(false).
			BorderForeground(subtle).
			MarginRight(2).
			Render

	listItem = lipgloss.NewStyle().PaddingLeft(2).Render

	activeBullet = lipgloss.NewStyle().SetString(emoji.BackhandIndexPointingRight.String()).
			Foreground(special).
			PaddingRight(1).
			String()

	currentBullet = lipgloss.NewStyle().SetString("●").
			Foreground(special).
			PaddingRight(1).
			String()

	crossmarkBullet = lipgloss.NewStyle().SetString(emoji.CrossMarkButton.String()).
			PaddingRight(1).
			String()

	checkmarkBullet = lipgloss.NewStyle().SetString(emoji.CheckMarkButton.String()).
			Foreground(special).
			PaddingRight(1).
			String()

	listDone = func(s string) string {
		return checkmarkBullet + s
	}

	listActive = func(s string) string {
		return activeBullet + s
	}

	listCross = func(s string) string {
		return crossmarkBullet + s
	}

	listBullet = func(s string) string {
		return currentBullet + s
	}

	// Paragraphs/History.

	historyStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Margin(1, 1, 0, 0).
			Padding(0, 0).
			Width(columnWidth)

	// Status Bar.

	statusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	orangeStatusPillStyle = lipgloss.NewStyle().
				Inherit(statusBarStyle).
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(lipgloss.Color("#FF5F87")).
				Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(0)

	encodingStyle = statusNugget.Copy().
			Background(lipgloss.Color("#A550DF")).
			Padding(0, 1).
			Align(lipgloss.Right)

	taskListStatusIncompleteStyle = statusNugget.Copy().
					Background(lipgloss.Color("#fff382")).
					Foreground(lipgloss.Color("#000000")).
					Padding(0, 1).
					Align(lipgloss.Right)

	taskListStatusCompleteStyle = statusNugget.Copy().
					Background(lipgloss.Color("#51d88a")).
					Padding(0, 1).
					Align(lipgloss.Right)

	statusText = lipgloss.NewStyle().Inherit(statusBarStyle).Padding(0, 1)

	purpleStatusPillStyle = statusNugget.Copy().Background(lipgloss.Color("#6124DF"))

	// Page.

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

func colorGrid(xSteps, ySteps int) [][]string {
	x0y0, _ := colorful.Hex("#F25D94")
	x1y0, _ := colorful.Hex("#EDFF82")
	x0y1, _ := colorful.Hex("#643AFF")
	x1y1, _ := colorful.Hex("#14F9D5")

	x0 := make([]colorful.Color, ySteps)
	for i := range x0 {
		x0[i] = x0y0.BlendLuv(x0y1, float64(i)/float64(ySteps))
	}

	x1 := make([]colorful.Color, ySteps)
	for i := range x1 {
		x1[i] = x1y0.BlendLuv(x1y1, float64(i)/float64(ySteps))
	}

	grid := make([][]string, ySteps)
	for x := 0; x < ySteps; x++ {
		y0 := x0[x]
		grid[x] = make([]string, xSteps)
		for y := 0; y < xSteps; y++ {
			grid[x][y] = y0.BlendLuv(x1[x], float64(y)/float64(xSteps)).Hex()
		}
	}

	return grid
}

func errorView(err error, fatal bool) string {
	exitMsg := "press any key to "
	if fatal {
		exitMsg += "exit"
	} else {
		exitMsg += "return"
	}
	s := fmt.Sprintf("%s\n\n%v\n\n%s",
		te.String(" ERROR ").
			Foreground(lib.Cream.Color()).
			Background(lib.Red.Color()).
			String(),
		err,
		common.Subtle(exitMsg),
	)
	return dialogBoxStyle.Copy().Align(lipgloss.Center).Render(s)
}

func helpView() string {
	b := strings.Builder{}
	for _, a := range controls {
		fmt.Fprintf(&b, "%s: %s\n", a.key, a.short)
	}
	return dialogBoxStyle.Align(lipgloss.Center).Render(b.String())
}
