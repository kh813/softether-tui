package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type bridgeFormField int

const (
	bridgeFieldHub bridgeFormField = iota
	bridgeFieldDevice
	bridgeFieldTap
	bridgeFormFieldCount
)

// bridgeForm is the local bridge creation screen: which hub to bridge, onto
// which physical device, and whether to use a tap device.
type bridgeForm struct {
	inputs [2]textinput.Model // hub name, device name
	tap    bool
	focus  bridgeFormField
}

func newBridgeForm() *bridgeForm {
	hub := textinput.New()
	hub.Placeholder = tr("Hub名")
	hub.CharLimit = 63

	device := textinput.New()
	device.Placeholder = tr("デバイス名 (上の一覧を参照)")
	device.CharLimit = 63

	f := &bridgeForm{inputs: [2]textinput.Model{hub, device}}
	f.setFocus(bridgeFieldHub)
	return f
}

func (f *bridgeForm) Reset() {
	f.inputs[bridgeFieldHub].SetValue("")
	f.inputs[bridgeFieldDevice].SetValue("")
	f.tap = false
	f.setFocus(bridgeFieldHub)
}

func (f *bridgeForm) IsDirty() bool {
	return strings.TrimSpace(f.inputs[0].Value()) != "" ||
		strings.TrimSpace(f.inputs[1].Value()) != "" ||
		f.tap
}

func (f *bridgeForm) setFocus(field bridgeFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

func (f *bridgeForm) Build() (hubName, deviceName string, tap bool, err error) {
	hubName = strings.TrimSpace(f.inputs[bridgeFieldHub].Value())
	deviceName = strings.TrimSpace(f.inputs[bridgeFieldDevice].Value())
	if hubName == "" {
		err = errors.New(tr("Hub名は必須です"))
		return
	}
	if deviceName == "" {
		err = errors.New(tr("デバイス名は必須です"))
		return
	}
	tap = f.tap
	return
}

func (f *bridgeForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % bridgeFormFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + bridgeFormFieldCount) % bridgeFormFieldCount)
		return nil
	case "left", "right":
		if f.focus == bridgeFieldTap {
			f.tap = !f.tap
			return nil
		}
	}

	if int(f.focus) < len(f.inputs) {
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
	return nil
}

func (f *bridgeForm) View() string {
	labels := []string{tr("Hub名"), tr("デバイス名")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("ローカルブリッジ作成")) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == bridgeFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-10s %s\n", marker, labels[i]+":", in.View())
	}

	tapMarker := "  "
	if f.focus == bridgeFieldTap {
		tapMarker = "> "
	}
	tapLabel := "no"
	if f.tap {
		tapLabel = "yes"
	}
	fmt.Fprintf(&b, "%s%-10s < %s >\n", tapMarker, "TAP:", tapLabel)

	b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  ←→: TAP切替  Enter: 作成  Esc: キャンセル")))
	return b.String()
}
