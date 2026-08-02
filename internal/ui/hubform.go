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

func (f *hubForm) IsDirty() bool {
	return strings.TrimSpace(f.inputs[0].Value()) != "" ||
		strings.TrimSpace(f.inputs[1].Value()) != ""
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
	password = strings.TrimSpace(f.inputs[hubFieldPassword].Value())
	if name == "" {
		return "", "", errors.New(tr("Hub名は必須です"))
	}
	if password == "" {
		return "", "", errors.New(tr("初期管理パスワードは必須です"))
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
	labels := []string{tr("Hub name"), tr("Admin password")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("Hub作成")) + "\n\n")
	for i, in := range f.inputs {
		marker := "  "
		if f.focus == hubFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", marker, labels[i]+":", in.View())
	}

	nameValid := strings.TrimSpace(f.inputs[hubFieldName].Value()) != ""
	pwValid := strings.TrimSpace(f.inputs[hubFieldPassword].Value()) != ""
	canSave := nameValid && pwValid

	b.WriteString("\n")
	if canSave {
		b.WriteString(saveKeyStyle.Render(" [ Save ] ") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Enter", tr("作成 (Save)"), "Esc", tr("キャンセル")))
	} else {
		b.WriteString(dimStyle.Render("  [ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Esc", tr("キャンセル")))
	}
	return b.String()
}
