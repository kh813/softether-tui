package ui

import (
	"strings"

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

	// Tab styles
	tabKeyStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1)
	tabSepStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241")).Background(lipgloss.Color("235"))

	// Key binding styles
	keyStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	saveKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1)
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
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

// renderHelp renders pairs of (key, action description) with distinct key color.
func renderHelp(pairs ...string) string {
	var parts []string
	for i := 0; i < len(pairs); i += 2 {
		k := pairs[i]
		d := ""
		if i+1 < len(pairs) {
			d = pairs[i+1]
		}
		style := keyStyle
		if strings.Contains(k, "Tab") || strings.Contains(k, "←/→") {
			style = tabKeyStyle
		} else if k == "s" || strings.Contains(strings.ToLower(d), "save") || strings.Contains(d, "保存") {
			style = saveKeyStyle
		}
		if d == "" {
			parts = append(parts, style.Render(k))
		} else {
			parts = append(parts, style.Render(k)+descStyle.Render(":"+d))
		}
	}
	return strings.Join(parts, "  ")
}
