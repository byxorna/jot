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
	StashItemLineSecondaryFocused   = dullFuchsiaFg
	StashItemLinePrimaryUnfocused   = brightGrayFg
	StashItemLineSecondaryUnfocused = dimBrightGrayFg

	normalFg    = NewFgStyle(lib.NewColorPair("#dddddd", "#1a1a1a"))
	dimNormalFg = NewFgStyle(lib.NewColorPair("#777777", "#A49FA5"))

	brightGrayFg    = NewFgStyle(lib.NewColorPair("#979797", "#847A85"))
	dimBrightGrayFg = NewFgStyle(lib.NewColorPair("#4D4D4D", "#C2B8C2"))

	grayFg     = NewFgStyle(lib.NewColorPair("#626262", "#909090"))
	midGrayFg  = NewFgStyle(lib.NewColorPair("#4A4A4A", "#B2B2B2"))
	darkGrayFg = NewFgStyle(lib.NewColorPair("#3C3C3C", "#DDDADA"))

	greenFg        = NewFgStyle(lib.NewColorPair("#04B575", "#04B575"))
	semiDimGreenFg = NewFgStyle(lib.NewColorPair("#036B46", "#35D79C"))
	dimGreenFg     = NewFgStyle(lib.NewColorPair("#0B5137", "#72D2B0"))

	FuchsiaFg    = NewFgStyle(lib.Fuschia)
	dimFuchsiaFg = NewFgStyle(lib.NewColorPair("#99519E", "#F1A8FF"))

	dullFuchsiaFg    = NewFgStyle(lib.NewColorPair("#AD58B4", "#F793FF"))
	dimDullFuchsiaFg = NewFgStyle(lib.NewColorPair("#6B3A6F", "#F6C9FF"))

	indigoFg    = NewFgStyle(lib.Indigo)
	dimIndigoFg = NewFgStyle(lib.NewColorPair("#494690", "#9498FF"))

	subtleIndigoFg    = NewFgStyle(lib.NewColorPair("#514DC1", "#7D79F6"))
	dimSubtleIndigoFg = NewFgStyle(lib.NewColorPair("#383584", "#BBBDFF"))

	yellowFg     = NewFgStyle(lib.YellowGreen)                        // renders light green on light backgrounds
	dullYellowFg = NewFgStyle(lib.NewColorPair("#9BA92F", "#6BCB94")) // renders light green on light backgrounds
	redFg        = NewFgStyle(lib.Red)
	faintRedFg   = NewFgStyle(lib.FaintRed)

	// Ultimately, we should transition to named styles
	//tabColor = NewFgStyle(lib.NewColorPair("#626262", "#909090"))
	//selectedTabColor = NewFgStyle(lib.NewColorPair("#979797", "#332F33"))
	tabColor         = instaPurple
	selectedTabColor = instaMagenta

	// shades of teal
	// https://www.color-hex.com/color-palette/4666
	teal1 = NewFgStyle(lib.NewColorPair("#b2d8d8", "#b2d8d8"))
	teal2 = NewFgStyle(lib.NewColorPair("#66b2b2", "#66b2b2"))
	teal3 = NewFgStyle(lib.NewColorPair("#008080", "#008080"))
	teal4 = NewFgStyle(lib.NewColorPair("#006666", "#006666"))
	teal5 = NewFgStyle(lib.NewColorPair("#004c4c", "#004c4c"))

	// instagram color palette
	// https://www.color-hex.com/color-palette/44340
	instaYellow  = NewFgStyle(lib.NewColorPair("#feda75", "#feda75"))
	instaOrange  = NewFgStyle(lib.NewColorPair("#fa7e1e", "#fa7e1e"))
	instaMagenta = NewFgStyle(lib.NewColorPair("#d62976", "#d62976"))
	instaPurple  = NewFgStyle(lib.NewColorPair("#962fbf", "#962fbf"))
	instaBlue    = NewFgStyle(lib.NewColorPair("#4f5bd5", "#4f5bd5"))
)

// Returns a termenv style with foreground and background options.
func newStyle(fg, bg lib.ColorPair, bold bool) func(string) string {
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
