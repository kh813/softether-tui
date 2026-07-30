package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type hubFormField int

const (
	hubFieldName hubFormField = iota
	hubFieldPassword
	hubFieldCount
)

// hubForm is the Hub creation screen (name + initial admin password).
type hubForm struct {
	inputs [2]textinput.Model
	focus  hubFormField
}

func newHubForm() *hubForm {
	name := textinput.New()
	name.Placeholder = tr("Hub名")
	name.CharLimit = 63

	pw := textinput.New()
	pw.Placeholder = tr("初期管理パスワード")
	pw.CharLimit = 128
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '*'

	f := &hubForm{inputs: [2]textinput.Model{name, pw}}
	f.setFocus(hubFieldName)
	return f
}

func (f *hubForm) Reset() {
	f.inputs[hubFieldName].SetValue("")
	f.inputs[hubFieldPassword].SetValue("")
	f.setFocus(hubFieldName)
}

func (f *hubForm) setFocus(field hubFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	f.inputs[field].Focus()
}

// Build validates the form and returns the hub name and initial password.
func (f *hubForm) Build() (name, password string, err error) {
	name = strings.TrimSpace(f.inputs[hubFieldName].Value())
	password = f.inputs[hubFieldPassword].Value()
	if name == "" {
		return "", "", errors.New(tr("Hub名は必須です"))
	}
	return name, password, nil
}

func (f *hubForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % hubFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + hubFieldCount) % hubFieldCount)
		return nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

func (f *hubForm) View() string {
	labels := []string{tr("Hub名"), tr("初期管理パスワード")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("Hub作成")) + "\n\n")
	for i, in := range f.inputs {
		marker := "  "
		if f.focus == hubFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-18s %s\n", marker, labels[i]+":", in.View())
	}
	b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  Enter: 作成  Esc: キャンセル")))
	return b.String()
}
