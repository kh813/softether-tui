package ui

import (
	"github.com/charmbracelet/lipgloss"

	"softether-tui/internal/i18n"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Bold(true)
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

// asciiBorder avoids Unicode box-drawing characters, which mojibake on the
// same non-UTF8-locale servers that motivated tr()/enCatalog.
var asciiBorder = lipgloss.Border{
	Top:         "-",
	Bottom:      "-",
	Left:        "|",
	Right:       "|",
	TopLeft:     "+",
	TopRight:    "+",
	BottomLeft:  "+",
	BottomRight: "+",
}

// borderStyle returns the panel border style, in Unicode (rounded corners)
// for Japanese or ASCII-safe for English/unset.
func borderStyle() lipgloss.Style {
	border := asciiBorder
	if lang == i18n.JA {
		border = lipgloss.RoundedBorder()
	}
	return lipgloss.NewStyle().Border(border).Padding(0, 1)
}
