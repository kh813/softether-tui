package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type groupFormField int

const (
	groupFieldName groupFormField = iota
	groupFieldRealName
	groupFieldNote
	groupFormFieldCount
)

type groupForm struct {
	inputs [3]textinput.Model
	focus  groupFormField
}

func newGroupForm() *groupForm {
	labels := []string{tr("グループ名"), tr("表示名 (任意)"), tr("備考 (任意)")}
	f := &groupForm{}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	f.setFocus(groupFieldName)
	return f
}

func (f *groupForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.setFocus(groupFieldName)
}

func (f *groupForm) setFocus(field groupFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	f.inputs[field].Focus()
}

func (f *groupForm) Build() (name string, opts vpncmd.GroupCreateOptions, err error) {
	name = strings.TrimSpace(f.inputs[groupFieldName].Value())
	if name == "" {
		err = errors.New(tr("グループ名は必須です"))
		return
	}
	opts = vpncmd.GroupCreateOptions{
		RealName: strings.TrimSpace(f.inputs[groupFieldRealName].Value()),
		Note:     strings.TrimSpace(f.inputs[groupFieldNote].Value()),
	}
	return
}

func (f *groupForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % groupFormFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + groupFormFieldCount) % groupFormFieldCount)
		return nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

func (f *groupForm) View() string {
	labels := []string{tr("グループ名"), tr("表示名"), tr("備考")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("グループ作成")) + "\n\n")
	for i, in := range f.inputs {
		marker := "  "
		if f.focus == groupFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-10s %s\n", marker, labels[i]+":", in.View())
	}
	b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  Enter: 作成  Esc: キャンセル")))
	return b.String()
}
