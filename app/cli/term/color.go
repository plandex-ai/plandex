package term

import (
	"github.com/fatih/color"
	"github.com/muesli/termenv"
)

var IsDarkBg = termenv.HasDarkBackground()

var ColorHiGreen color.Attribute
var ColorHiMagenta color.Attribute
var ColorHiRed color.Attribute
var ColorHiYellow color.Attribute
var ColorHiCyan color.Attribute
var ColorHiBlue color.Attribute

func init() {

	if IsDarkBg {
		ColorHiGreen = color.FgHiGreen
		ColorHiMagenta = color.FgHiMagenta
		ColorHiRed = color.FgHiRed
		ColorHiYellow = color.FgHiYellow
		ColorHiCyan = color.FgHiCyan
		ColorHiBlue = color.FgHiBlue
	} else {
		ColorHiGreen = color.FgGreen
		ColorHiMagenta = color.FgMagenta
		ColorHiRed = color.FgRed
		ColorHiYellow = color.FgYellow
		ColorHiCyan = color.FgCyan
		ColorHiBlue = color.FgBlue
	}
}
