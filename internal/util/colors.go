package util

import (
	"github.com/fatih/color"
)

// ColorMap maps color names to terminal color attributes
var ColorMap = map[string]color.Attribute{
	"black":   color.FgBlack,
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,
	"brightred":     color.FgHiRed,
	"brightgreen":   color.FgHiGreen,
	"brightyellow":  color.FgHiYellow,
	"brightblue":    color.FgHiBlue,
	"brightmagenta": color.FgHiMagenta,
	"brightcyan":    color.FgHiCyan,
	"brightwhite":   color.FgHiWhite,
}

// GetTerminalColor returns a terminal color based on a color name
func GetTerminalColor(colorName string, defaultColor color.Attribute) *color.Color {
	if attr, ok := ColorMap[colorName]; ok {
		return color.New(attr)
	}
	return color.New(defaultColor)
}
