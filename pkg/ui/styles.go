package ui

import (
	lib "github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

type StyleFunc func(string) string

const (
	darkGrayHex = "#333333"
)

var (

	// Stash Item Colors
	StashItemLinePrimaryFocused     = FuchsiaFg
	StashItemLineSecondaryFocused   = DullFuchsiaFg
	StashItemLinePrimaryUnfocused   = BrightGrayFg
	StashItemLineSecondaryUnfocused = DimBrightGrayFg

	NormalFg    = NewFgStyle(lib.NewColorPair("#dddddd", "#1a1a1a"))
	DimNormalFg = NewFgStyle(lib.NewColorPair("#777777", "#A49FA5"))

	BrightGrayFg    = NewFgStyle(lib.NewColorPair("#979797", "#847A85"))
	DimBrightGrayFg = NewFgStyle(lib.NewColorPair("#4D4D4D", "#C2B8C2"))

	GrayFg     = NewFgStyle(lib.NewColorPair("#626262", "#909090"))
	MidGrayFg  = NewFgStyle(lib.NewColorPair("#4A4A4A", "#B2B2B2"))
	DarkGrayFg = NewFgStyle(lib.NewColorPair("#3C3C3C", "#DDDADA"))

	GreenFg        = NewFgStyle(lib.NewColorPair("#04B575", "#04B575"))
	SemiDimGreenFg = NewFgStyle(lib.NewColorPair("#036B46", "#35D79C"))
	DimGreenFg     = NewFgStyle(lib.NewColorPair("#0B5137", "#72D2B0"))

	FuchsiaFg    = NewFgStyle(lib.Fuschia)
	DimFuchsiaFg = NewFgStyle(lib.NewColorPair("#99519E", "#F1A8FF"))

	DullFuchsiaFg    = NewFgStyle(lib.NewColorPair("#AD58B4", "#F793FF"))
	DimDullFuchsiaFg = NewFgStyle(lib.NewColorPair("#6B3A6F", "#F6C9FF"))

	IndigoFg    = NewFgStyle(lib.Indigo)
	DimIndigoFg = NewFgStyle(lib.NewColorPair("#494690", "#9498FF"))

	SubtleIndigoFg    = NewFgStyle(lib.NewColorPair("#514DC1", "#7D79F6"))
	DimSubtleIndigoFg = NewFgStyle(lib.NewColorPair("#383584", "#BBBDFF"))

	YellowFg     = NewFgStyle(lib.YellowGreen)                        // renders light green on light backgrounds
	DullYellowFg = NewFgStyle(lib.NewColorPair("#9BA92F", "#6BCB94")) // renders light green on light backgrounds
	RedFg        = NewFgStyle(lib.Red)
	FaintRedFg   = NewFgStyle(lib.FaintRed)

	// Ultimately, we should transition to named styles
	//tabColor = NewFgStyle(lib.NewColorPair("#626262", "#909090"))
	//selectedTabColor = NewFgStyle(lib.NewColorPair("#979797", "#332F33"))
	TabColor         = InstaPurple
	SelectedTabColor = InstaMagenta

	// shades of teal
	// https://www.color-hex.com/color-palette/4666
	Teal1 = NewFgStyle(lib.NewColorPair("#b2d8d8", "#b2d8d8"))
	Teal2 = NewFgStyle(lib.NewColorPair("#66b2b2", "#66b2b2"))
	Teal3 = NewFgStyle(lib.NewColorPair("#008080", "#008080"))
	Teal4 = NewFgStyle(lib.NewColorPair("#006666", "#006666"))
	Teal5 = NewFgStyle(lib.NewColorPair("#004c4c", "#004c4c"))

	// instagram color palette
	// https://www.color-hex.com/color-palette/44340
	InstaYellow  = NewFgStyle(lib.NewColorPair("#feda75", "#feda75"))
	InstaOrange  = NewFgStyle(lib.NewColorPair("#fa7e1e", "#fa7e1e"))
	InstaMagenta = NewFgStyle(lib.NewColorPair("#d62976", "#d62976"))
	InstaPurple  = NewFgStyle(lib.NewColorPair("#962fbf", "#962fbf"))
	InstaBlue    = NewFgStyle(lib.NewColorPair("#4f5bd5", "#4f5bd5"))
)

// Returns a termenv style with foreground and background options.
func NewStyle(fg, bg lib.ColorPair, bold bool) func(string) string {
	s := te.Style{}.Foreground(fg.Color()).Background(bg.Color())
	if bold {
		s = s.Bold()
	}
	return s.Styled
}

// Returns a new termenv style with background options only.
func NewFgStyle(c lib.ColorPair) StyleFunc {
	return te.Style{}.Foreground(c.Color()).Styled
}
