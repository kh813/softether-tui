package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type userFormField int

const (
	userFieldName userFormField = iota
	userFieldGroup
	userFieldRealName
	userFieldNote
	userFieldAuthType
	userFieldPassword
	userFormFieldCount
)

var userAuthLabels = map[vpncmd.UserAuthType]string{
	vpncmd.UserAuthPassword:  "Password",
	vpncmd.UserAuthAnonymous: "Anonymous",
	vpncmd.UserAuthRadius:    "Radius",
}

var userAuthOrder = []vpncmd.UserAuthType{
	vpncmd.UserAuthPassword, vpncmd.UserAuthAnonymous, vpncmd.UserAuthRadius,
}

// userForm is the user creation screen. NTLM and certificate authentication
// are intentionally not offered: their vpncmd parameter names are
// unconfirmed (see vpncmd_commands.md).
type userForm struct {
	inputs      [4]textinput.Model // name, group, realname, note
	password    textinput.Model
	authType    vpncmd.UserAuthType
	focus       userFormField
	groups      []string // available groups for selector
	groupIndex  int      // 0 for (none), 1..len(groups)
}

func newUserForm() *userForm {
	labels := []string{tr("ユーザー名"), tr("グループ (任意)"), tr("表示名 (任意)"), tr("備考 (任意)")}
	f := &userForm{}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	pw := textinput.New()
	pw.Placeholder = tr("パスワード")
	pw.CharLimit = 128
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '*'
	f.password = pw
	f.setFocus(userFieldName)
	return f
}

func (f *userForm) SetGroups(groups []string) {
	f.groups = groups
	f.groupIndex = 0
}

func (f *userForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.password.SetValue("")
	f.authType = vpncmd.UserAuthPassword
	f.groupIndex = 0
	f.setFocus(userFieldName)
}

func (f *userForm) setFocus(field userFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.password.Blur()
	f.focus = field
	if field == userFieldPassword {
		f.password.Focus()
	} else if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

// fieldCount returns the number of focusable fields: the password field is
// only reachable when Password authentication is selected.
func (f *userForm) fieldCount() userFormField {
	if f.authType == vpncmd.UserAuthPassword {
		return userFormFieldCount
	}
	return userFieldPassword
}

// Build validates the form and returns the pieces needed to create the user:
// the base UserCreate fields, plus the auth method to apply afterward.
func (f *userForm) Build() (name string, opts vpncmd.UserCreateOptions, authType vpncmd.UserAuthType, password string, err error) {
	name = strings.TrimSpace(f.inputs[userFieldName].Value())
	if name == "" {
		err = errors.New(tr("ユーザー名は必須です"))
		return
	}
	groupVal := ""
	if f.groupIndex > 0 && f.groupIndex <= len(f.groups) {
		groupVal = f.groups[f.groupIndex-1]
	} else {
		groupVal = strings.TrimSpace(f.inputs[userFieldGroup].Value())
	}
	opts = vpncmd.UserCreateOptions{
		Group:    groupVal,
		RealName: strings.TrimSpace(f.inputs[userFieldRealName].Value()),
		Note:     strings.TrimSpace(f.inputs[userFieldNote].Value()),
	}
	authType = f.authType
	password = f.password.Value()
	if authType == vpncmd.UserAuthPassword && password == "" {
		err = errors.New(tr("Password認証を選択した場合はパスワードが必須です"))
	}
	return
}

func (f *userForm) Update(msg tea.KeyMsg) tea.Cmd {
	count := f.fieldCount()
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % count)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + count) % count)
		return nil
	case "left", "right":
		if f.focus == userFieldGroup && len(f.groups) > 0 {
			f.cycleGroup(msg.String() == "right")
			return nil
		}
		if f.focus == userFieldAuthType {
			f.cycleAuthType(msg.String() == "right")
			return nil
		}
	}

	switch f.focus {
	case userFieldPassword:
		var cmd tea.Cmd
		f.password, cmd = f.password.Update(msg)
		return cmd
	case userFieldAuthType:
		return nil
	default:
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
}

func (f *userForm) cycleGroup(forward bool) {
	total := len(f.groups) + 1
	if forward {
		f.groupIndex = (f.groupIndex + 1) % total
	} else {
		f.groupIndex = (f.groupIndex - 1 + total) % total
	}
}

func (f *userForm) cycleAuthType(forward bool) {
	idx := 0
	for i, a := range userAuthOrder {
		if a == f.authType {
			idx = i
		}
	}
	if forward {
		idx = (idx + 1) % len(userAuthOrder)
	} else {
		idx = (idx - 1 + len(userAuthOrder)) % len(userAuthOrder)
	}
	f.authType = userAuthOrder[idx]
}

func (f *userForm) View() string {
	labels := []string{tr("ユーザー名"), tr("グループ"), tr("表示名"), tr("備考")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("ユーザー作成")) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == userFormField(i) {
			marker = "> "
		}
		if userFormField(i) == userFieldGroup && len(f.groups) > 0 {
			groupLabel := "(なし)"
			if f.groupIndex > 0 && f.groupIndex <= len(f.groups) {
				groupLabel = f.groups[f.groupIndex-1]
			}
			fmt.Fprintf(&b, "%s%-14s < %s >\n", marker, labels[i]+":", groupLabel)
		} else {
			fmt.Fprintf(&b, "%s%-14s %s\n", marker, labels[i]+":", in.View())
		}
	}

	authMarker := "  "
	if f.focus == userFieldAuthType {
		authMarker = "> "
	}
	fmt.Fprintf(&b, "%s%-14s < %s >\n", authMarker, tr("認証方式:"), userAuthLabels[f.authType])

	if f.authType == vpncmd.UserAuthPassword {
		pwMarker := "  "
		if f.focus == userFieldPassword {
			pwMarker = "> "
		}
		fmt.Fprintf(&b, "%s%-14s %s\n", pwMarker, tr("パスワード:"), f.password.View())
	}

	nameValid := strings.TrimSpace(f.inputs[userFieldName].Value()) != ""
	pwValid := f.authType != vpncmd.UserAuthPassword || strings.TrimSpace(f.password.Value()) != ""
	canSave := nameValid && pwValid

	b.WriteString("\n")
	if canSave {
		b.WriteString(saveKeyStyle.Render(" [ Save ] ") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "←→", tr("認証方式切替"), "Enter", tr("作成 (Save)"), "Esc", tr("キャンセル")))
	} else {
		b.WriteString(dimStyle.Render("  [ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "←→", tr("認証方式切替"), "Esc", tr("キャンセル")))
	}
	return b.String()
}
