package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type cascadeFormField int

const (
	cascadeFieldName cascadeFormField = iota
	cascadeFieldHost
	cascadeFieldPort
	cascadeFieldHub
	cascadeFieldUser
	cascadeFieldPassword
	cascadeFieldSave
	cascadeFieldCount
)

var cascadeInputCount = int(cascadeFieldSave)

type cascadeForm struct {
	inputs   [6]textinput.Model // name, host, port, hub, user, password
	focus    cascadeFormField
	editing  bool
	original string
}

func newCascadeForm() *cascadeForm {
	placeholders := []string{
		tr("接続設定名"),
		tr("接続先ホスト (例: 192.168.1.10)"),
		"443",
		tr("接続先 Hub 名"),
		tr("ユーザー名"),
		tr("パスワード (任意)"),
	}
	f := &cascadeForm{}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 128
		if i == 5 {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '*'
		}
		f.inputs[i] = ti
	}
	f.inputs[2].SetValue("443")
	f.setFocus(cascadeFieldName)
	return f
}

func (f *cascadeForm) Reset() {
	f.editing = false
	f.original = ""
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.inputs[2].SetValue("443")
	f.setFocus(cascadeFieldName)
}

func (f *cascadeForm) LoadCascade(name, host string, port int, hub, user string) {
	f.editing = true
	f.original = name
	f.inputs[0].SetValue(name)
	f.inputs[1].SetValue(host)
	f.inputs[2].SetValue(fmt.Sprintf("%d", port))
	f.inputs[3].SetValue(hub)
	f.inputs[4].SetValue(user)
	f.inputs[5].SetValue("")
	f.setFocus(cascadeFieldName)
}

func (f *cascadeForm) setFocus(field cascadeFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

func (f *cascadeForm) Build() (name string, opts vpncmd.CascadeCreateOptions, password string, err error) {
	name = strings.TrimSpace(f.inputs[cascadeFieldName].Value())
	if name == "" {
		err = errors.New(tr("接続設定名は必須です"))
		return
	}
	host := strings.TrimSpace(f.inputs[cascadeFieldHost].Value())
	if host == "" {
		err = errors.New(tr("接続先ホストは必須です"))
		return
	}
	portStr := strings.TrimSpace(f.inputs[cascadeFieldPort].Value())
	var port int
	if _, convErr := fmt.Sscanf(portStr, "%d", &port); convErr != nil || port <= 0 || port > 65535 {
		err = fmt.Errorf(tr("ポート番号が不正です: %q"), portStr)
		return
	}
	hub := strings.TrimSpace(f.inputs[cascadeFieldHub].Value())
	if hub == "" {
		err = errors.New(tr("接続先 Hub は必須です"))
		return
	}
	user := strings.TrimSpace(f.inputs[cascadeFieldUser].Value())
	if user == "" {
		err = errors.New(tr("ユーザー名は必須です"))
		return
	}
	password = f.inputs[cascadeFieldPassword].Value()

	opts = vpncmd.CascadeCreateOptions{
		ServerHost: host,
		ServerPort: port,
		Hub:        hub,
		User:       user,
	}
	return
}

func (f *cascadeForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % cascadeFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + cascadeFieldCount) % cascadeFieldCount)
		return nil
	}

	if int(f.focus) < cascadeInputCount {
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
	return nil
}

func (f *cascadeForm) View() string {
	labels := []string{
		tr("接続設定名"),
		tr("接続先ホスト"),
		tr("ポート"),
		tr("接続先 Hub"),
		tr("ユーザー名"),
		tr("パスワード"),
	}

	maxLen := 0
	for _, l := range labels {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	maxLen += 1

	var b strings.Builder
	title := tr("カスケード接続作成")
	if f.editing {
		title = tr("カスケード接続編集: ") + f.original
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == cascadeFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-*s %s\n", marker, maxLen, labels[i]+":", in.View())
	}

	nameValid := strings.TrimSpace(f.inputs[cascadeFieldName].Value()) != ""
	hostValid := strings.TrimSpace(f.inputs[cascadeFieldHost].Value()) != ""
	portValid := strings.TrimSpace(f.inputs[cascadeFieldPort].Value()) != ""
	hubValid := strings.TrimSpace(f.inputs[cascadeFieldHub].Value()) != ""
	userValid := strings.TrimSpace(f.inputs[cascadeFieldUser].Value()) != ""
	canSave := nameValid && hostValid && portValid && hubValid && userValid

	b.WriteString("\n")
	saveMarker := "  "
	if f.focus == cascadeFieldSave {
		saveMarker = "> "
	}

	if canSave {
		if f.focus == cascadeFieldSave {
			b.WriteString(saveMarker + saveKeyStyle.Render(" [ Save ] ") + "\n")
		} else {
			b.WriteString(saveMarker + inactiveTabStyle.Render(" [ Save ] ") + "\n")
		}
		b.WriteString("\n" + renderHelp(
			"Tab/↑↓", tr("項目移動"),
			"Enter", tr("保存 (Save)"),
			"Esc", tr("キャンセル"),
		))
	} else {
		b.WriteString(saveMarker + dimStyle.Render("[ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp(
			"Tab/↑↓", tr("項目移動"),
			"Esc", tr("キャンセル"),
		))
	}
	return b.String()
}
