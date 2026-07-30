package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type accountFormField int

const (
	accountFieldName accountFormField = iota
	accountFieldServerHost
	accountFieldServerPort
	accountFieldHub
	accountFieldAuthType
	accountFieldPassword
	accountFormFieldCount
)

var accountAuthLabels = map[vpncmd.AccountAuthType]string{
	vpncmd.AccountAuthPassword:  "Password",
	vpncmd.AccountAuthAnonymous: "Anonymous",
}

var accountAuthOrder = []vpncmd.AccountAuthType{
	vpncmd.AccountAuthPassword, vpncmd.AccountAuthAnonymous,
}

// accountForm is the VPN Client connection-account creation screen.
// Certificate authentication is intentionally not offered (see
// vpncmd.AccountAuthType).
type accountForm struct {
	inputs   [4]textinput.Model // name, server host, server port, hub
	password textinput.Model
	authType vpncmd.AccountAuthType
	focus    accountFormField
}

func newAccountForm() *accountForm {
	labels := []string{tr("接続名"), tr("サーバーホスト"), tr("ポート"), tr("接続先 Hub")}
	f := &accountForm{}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	f.inputs[accountFieldServerPort].SetValue("443")
	pw := textinput.New()
	pw.Placeholder = tr("パスワード")
	pw.CharLimit = 128
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '*'
	f.password = pw
	f.setFocus(accountFieldName)
	return f
}

func (f *accountForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.inputs[accountFieldServerPort].SetValue("443")
	f.password.SetValue("")
	f.authType = vpncmd.AccountAuthPassword
	f.setFocus(accountFieldName)
}

func (f *accountForm) setFocus(field accountFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.password.Blur()
	f.focus = field
	if field == accountFieldPassword {
		f.password.Focus()
	} else if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

// fieldCount returns the number of focusable fields: the password field is
// only reachable when Password authentication is selected.
func (f *accountForm) fieldCount() accountFormField {
	if f.authType == vpncmd.AccountAuthPassword {
		return accountFormFieldCount
	}
	return accountFieldPassword
}

// Build validates the form and returns the pieces needed to create the
// account: the base AccountCreate fields, plus the auth method to apply
// afterward (mirrors userForm.Build).
func (f *accountForm) Build() (name string, opts vpncmd.AccountCreateOptions, authType vpncmd.AccountAuthType, password string, err error) {
	name = strings.TrimSpace(f.inputs[accountFieldName].Value())
	if name == "" {
		err = errors.New(tr("接続名は必須です"))
		return
	}
	host := strings.TrimSpace(f.inputs[accountFieldServerHost].Value())
	if host == "" {
		err = errors.New(tr("サーバーホストは必須です"))
		return
	}
	portStr := strings.TrimSpace(f.inputs[accountFieldServerPort].Value())
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil || port <= 0 || port > 65535 {
		err = fmt.Errorf(tr("ポート番号が不正です: %q"), portStr)
		return
	}
	hub := strings.TrimSpace(f.inputs[accountFieldHub].Value())
	if hub == "" {
		err = errors.New(tr("接続先 Hub は必須です"))
		return
	}

	opts = vpncmd.AccountCreateOptions{ServerHost: host, ServerPort: port, Hub: hub}
	authType = f.authType
	password = f.password.Value()
	if authType == vpncmd.AccountAuthPassword && password == "" {
		err = errors.New(tr("Password認証を選択した場合はパスワードが必須です"))
	}
	return
}

func (f *accountForm) Update(msg tea.KeyMsg) tea.Cmd {
	count := f.fieldCount()
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % count)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + count) % count)
		return nil
	case "left", "right":
		if f.focus == accountFieldAuthType {
			f.cycleAuthType(msg.String() == "right")
			return nil
		}
	}

	switch f.focus {
	case accountFieldPassword:
		var cmd tea.Cmd
		f.password, cmd = f.password.Update(msg)
		return cmd
	case accountFieldAuthType:
		return nil
	default:
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
}

func (f *accountForm) cycleAuthType(forward bool) {
	idx := 0
	for i, a := range accountAuthOrder {
		if a == f.authType {
			idx = i
		}
	}
	if forward {
		idx = (idx + 1) % len(accountAuthOrder)
	} else {
		idx = (idx - 1 + len(accountAuthOrder)) % len(accountAuthOrder)
	}
	f.authType = accountAuthOrder[idx]
}

func (f *accountForm) View() string {
	labels := []string{tr("接続名"), tr("サーバーホスト"), tr("ポート"), tr("接続先 Hub")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("VPN Client 接続作成")) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == accountFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-14s %s\n", marker, labels[i]+":", in.View())
	}

	authMarker := "  "
	if f.focus == accountFieldAuthType {
		authMarker = "> "
	}
	fmt.Fprintf(&b, "%s%-14s < %s >\n", authMarker, tr("認証方式:"), accountAuthLabels[f.authType])

	if f.authType == vpncmd.AccountAuthPassword {
		pwMarker := "  "
		if f.focus == accountFieldPassword {
			pwMarker = "> "
		}
		fmt.Fprintf(&b, "%s%-14s %s\n", pwMarker, tr("パスワード:"), f.password.View())
	}

	b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  ←→: 認証方式切替  Enter: 作成  Esc: キャンセル")))
	b.WriteString("\n" + dimStyle.Render(tr("(証明書認証は未対応)")))
	return b.String()
}
