package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type acFormField int

const (
	acFieldAction acFormField = iota
	acFieldPriority
	acFieldIPMask
	acFieldSave
	acFormFieldCount
)

type acForm struct {
	pass   bool
	inputs [2]textinput.Model // priority, ipMask
	focus  acFormField
}

func newAcForm() *acForm {
	priority := textinput.New()
	priority.Placeholder = "100"
	priority.SetValue("100")
	priority.CharLimit = 10

	ipMask := textinput.New()
	ipMask.Placeholder = tr("例: 192.168.1.0/24 または 10.0.0.1")
	ipMask.CharLimit = 64

	f := &acForm{
		pass:   false, // default Deny
		inputs: [2]textinput.Model{priority, ipMask},
	}
	f.setFocus(acFieldAction)
	return f
}

func (f *acForm) Reset() {
	f.pass = false
	f.inputs[0].SetValue("100")
	f.inputs[1].SetValue("")
	f.setFocus(acFieldAction)
}

func (f *acForm) setFocus(field acFormField) {
	f.focus = field
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	if field == acFieldPriority {
		f.inputs[0].Focus()
	} else if field == acFieldIPMask {
		f.inputs[1].Focus()
	}
}

func (f *acForm) NextField() {
	f.setFocus((f.focus + 1) % acFormFieldCount)
}

func (f *acForm) PrevField() {
	f.setFocus((f.focus - 1 + acFormFieldCount) % acFormFieldCount)
}

func (f *acForm) Build() (pass bool, priority int, ipMask string, err error) {
	pass = f.pass
	pStr := strings.TrimSpace(f.inputs[0].Value())
	if pStr == "" {
		pStr = "100"
	}
	p, err := strconv.Atoi(pStr)
	if err != nil || p <= 0 {
		return false, 0, "", errors.New(tr("優先度は正の整数を入力してください"))
	}
	priority = p

	ip := strings.TrimSpace(f.inputs[1].Value())
	if ip == "" {
		return false, 0, "", errors.New(tr("IPアドレス/サブネットは必須です"))
	}
	ipMask = ip
	return pass, priority, ipMask, nil
}

func (f *acForm) Update(msg tea.Msg) (*acForm, tea.Cmd) {
	if f.focus == acFieldPriority {
		var cmd tea.Cmd
		f.inputs[0], cmd = f.inputs[0].Update(msg)
		return f, cmd
	} else if f.focus == acFieldIPMask {
		var cmd tea.Cmd
		f.inputs[1], cmd = f.inputs[1].Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f *acForm) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("接続元IP制限ルールの追加")) + "\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	// 1. Action row
	cursor := "  "
	if f.focus == acFieldAction {
		cursor = "> "
	}
	actBadge := statusBarStyle.Render("[ " + tr("拒否 (Deny)") + " ]")
	if f.pass {
		actBadge = selectedStyle.Render("[ " + tr("許可 (Pass)") + " ]")
	}
	fmt.Fprintf(&b, "%s%-24s: %s\n", cursor, tr("動作 (Action)"), actBadge)

	// 2. Priority row
	cursor = "  "
	if f.focus == acFieldPriority {
		cursor = "> "
	}
	fmt.Fprintf(&b, "%s%-24s: %s\n", cursor, tr("優先度 (Priority)"), f.inputs[0].View())

	// 3. IP / Subnet row
	cursor = "  "
	if f.focus == acFieldIPMask {
		cursor = "> "
	}
	fmt.Fprintf(&b, "%s%-24s: %s\n", cursor, tr("IP/サブネット"), f.inputs[1].View())

	// 4. Save button
	b.WriteString("\n")
	cursor = "  "
	btnStyle := statusBarStyle
	if f.focus == acFieldSave {
		cursor = "> "
		btnStyle = selectedStyle
	}
	b.WriteString(cursor + btnStyle.Render("  [ "+tr("追加 (Save)")+" ]  ") + "\n")

	b.WriteString("\n" + renderHelp(
		"Tab/↑↓", tr("項目移動"),
		"Space/Enter", tr("動作切替/決定"),
		"Esc", tr("キャンセル"),
	))
	return b.String()
}
