package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// promptKind identifies which single-value action a prompt overlay collects
// input for. Like confirmDialog, this stores an action descriptor rather
// than a closure: Model uses value semantics, so a closure captured at Show
// time would see stale state by the time the user submits.
type promptKind int

const (
	promptNone promptKind = iota
	promptUserPassword
	promptUserGroup
	promptUserExpires
	promptListenerCreate
	promptAccountPassword
	promptConnectPassword
	promptInitialPassword
)

// prompt is a single-line input overlay, e.g. for password reset or setting
// a user's expiration date.
type prompt struct {
	active bool
	kind   promptKind
	target string
	label  string
	input  textinput.Model
}

func (p *prompt) Show(kind promptKind, target, label, placeholder string, mask bool) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	if mask {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	ti.Focus()
	*p = prompt{active: true, kind: kind, target: target, label: label, input: ti}
}

func (p *prompt) Hide() {
	*p = prompt{}
}

func (p *prompt) Update(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd
}

func (p *prompt) View() string {
	content := p.label + "\n\n" + p.input.View() + "\n\n" + dimStyle.Render(tr("Enter: 実行  Esc: キャンセル"))
	return borderStyle().BorderForeground(lipgloss.Color("39")).Render(content)
}
