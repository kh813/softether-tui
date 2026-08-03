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
	userFieldExpires
	userFieldAuthType
	userFieldAuthParam1 // Password / Cert Path / SignedCert CN / NTLM Alias
	userFieldAuthParam2 // SignedCert Serial
	userFormFieldCount
)

var userAuthLabels = map[vpncmd.UserAuthType]string{
	vpncmd.UserAuthPassword:   "Password",
	vpncmd.UserAuthAnonymous:  "Anonymous",
	vpncmd.UserAuthRadius:     "Radius",
	vpncmd.UserAuthCert:       "Cert (X.509)",
	vpncmd.UserAuthSignedCert: "Signed Cert (CA)",
	vpncmd.UserAuthNTLM:       "NTLM (Domain)",
}

var userAuthOrder = []vpncmd.UserAuthType{
	vpncmd.UserAuthPassword,
	vpncmd.UserAuthAnonymous,
	vpncmd.UserAuthRadius,
	vpncmd.UserAuthCert,
	vpncmd.UserAuthSignedCert,
	vpncmd.UserAuthNTLM,
}

type userForm struct {
	inputs         [5]textinput.Model // name, group, realname, note, expires
	authParam1     textinput.Model    // Password / Cert Path / SignedCert CN / NTLM Alias / Radius Alias
	authParam2     textinput.Model    // SignedCert Serial
	authType       vpncmd.UserAuthType
	focus          userFormField
	groups         []string // available groups for selector
	groupIndex     int      // 0 for (none), 1..len(groups)
	dropdownActive bool
	dropdownCursor int
	dropdownIsAuth bool // false = Group dropdown, true = Auth dropdown
}

func newUserForm() *userForm {
	labels := []string{tr("ユーザー名"), tr("グループ (任意)"), tr("表示名 (任意)"), tr("備考 (任意)"), tr("有効期限 (任意: YYYY/MM/DD)")}
	f := &userForm{}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	p1 := textinput.New()
	p1.CharLimit = 256
	f.authParam1 = p1

	p2 := textinput.New()
	p2.CharLimit = 256
	f.authParam2 = p2

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
	f.authParam1.SetValue("")
	f.authParam2.SetValue("")
	f.authType = vpncmd.UserAuthPassword
	f.groupIndex = 0
	f.dropdownActive = false
	f.setFocus(userFieldName)
}

func (f *userForm) setFocus(field userFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.authParam1.Blur()
	f.authParam2.Blur()
	f.focus = field
	if field == userFieldAuthParam1 {
		f.authParam1.Focus()
	} else if field == userFieldAuthParam2 {
		f.authParam2.Focus()
	} else if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

func (f *userForm) IsDirty() bool {
	return strings.TrimSpace(f.inputs[userFieldName].Value()) != "" ||
		strings.TrimSpace(f.inputs[userFieldRealName].Value()) != "" ||
		strings.TrimSpace(f.inputs[userFieldNote].Value()) != "" ||
		strings.TrimSpace(f.authParam1.Value()) != "" ||
		strings.TrimSpace(f.authParam2.Value()) != ""
}

func (f *userForm) fieldCount() userFormField {
	switch f.authType {
	case vpncmd.UserAuthPassword, vpncmd.UserAuthCert, vpncmd.UserAuthNTLM, vpncmd.UserAuthRadius:
		return userFieldAuthParam1 + 1
	case vpncmd.UserAuthSignedCert:
		return userFormFieldCount
	default:
		return userFieldAuthType + 1
	}
}

// Build validates the form and returns the pieces needed to create the user:
// the base UserCreate fields, plus auth parameters.
func (f *userForm) Build() (name string, opts vpncmd.UserCreateOptions, authType vpncmd.UserAuthType, param1, param2, expires string, err error) {
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
	param1 = strings.TrimSpace(f.authParam1.Value())
	param2 = strings.TrimSpace(f.authParam2.Value())

	if authType == vpncmd.UserAuthPassword && param1 == "" {
		err = errors.New(tr("Password認証を選択した場合はパスワードが必須です"))
		return
	}
	if authType == vpncmd.UserAuthCert && param1 == "" {
		err = errors.New(tr("Cert認証を選択した場合は証明書ファイルパスが必須です"))
		return
	}

	rawExpires := strings.TrimSpace(f.inputs[userFieldExpires].Value())
	if rawExpires != "" {
		formatted := normalizeAndFormatDate(rawExpires)
		if formatted == "" {
			err = errors.New(tr("有効期限は YYYY/MM/DD 形式（数字8桁等）で入力してください"))
			return
		}
		expires = formatted
	}
	return
}

func (f *userForm) Update(msg tea.KeyMsg) tea.Cmd {
	if f.dropdownActive {
		switch msg.String() {
		case "up", "k":
			if f.dropdownCursor > 0 {
				f.dropdownCursor--
			}
		case "down", "j":
			max := len(f.groups)
			if f.dropdownIsAuth {
				max = len(userAuthOrder) - 1
			}
			if f.dropdownCursor < max {
				f.dropdownCursor++
			}
		case "enter":
			if f.dropdownIsAuth {
				f.authType = userAuthOrder[f.dropdownCursor]
				if f.focus >= userFieldAuthParam1 {
					f.setFocus(userFieldAuthParam1)
				}
			} else {
				f.groupIndex = f.dropdownCursor
			}
			f.dropdownActive = false
		case "esc":
			f.dropdownActive = false
		}
		return nil
	}

	count := f.fieldCount()
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % count)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + count) % count)
		return nil
	case "enter":
		if f.focus == userFieldGroup && len(f.groups) > 0 {
			f.dropdownActive = true
			f.dropdownIsAuth = false
			f.dropdownCursor = f.groupIndex
			return nil
		}
		if f.focus == userFieldAuthType {
			f.dropdownActive = true
			f.dropdownIsAuth = true
			for i, a := range userAuthOrder {
				if a == f.authType {
					f.dropdownCursor = i
					break
				}
			}
			return nil
		}
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
	case userFieldAuthParam1:
		var cmd tea.Cmd
		f.authParam1, cmd = f.authParam1.Update(msg)
		return cmd
	case userFieldAuthParam2:
		var cmd tea.Cmd
		f.authParam2, cmd = f.authParam2.Update(msg)
		return cmd
	case userFieldAuthType:
		return nil
	default:
		if int(f.focus) < len(f.inputs) {
			var cmd tea.Cmd
			f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
			return cmd
		}
		return nil
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
	if f.focus >= userFieldAuthParam1 {
		f.setFocus(userFieldAuthParam1)
	}
}

func (f *userForm) View() string {
	labels := []string{tr("ユーザー名"), tr("グループ"), tr("表示名"), tr("備考"), tr("有効期限")}
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

	switch f.authType {
	case vpncmd.UserAuthPassword:
		pMarker := "  "
		if f.focus == userFieldAuthParam1 {
			pMarker = "> "
		}
		f.authParam1.EchoMode = textinput.EchoPassword
		f.authParam1.EchoCharacter = '*'
		f.authParam1.Placeholder = tr("パスワード")
		fmt.Fprintf(&b, "%s%-14s %s\n", pMarker, tr("パスワード:"), f.authParam1.View())
	case vpncmd.UserAuthCert:
		pMarker := "  "
		if f.focus == userFieldAuthParam1 {
			pMarker = "> "
		}
		f.authParam1.EchoMode = textinput.EchoNormal
		f.authParam1.Placeholder = tr("証明書ファイルパス (/path/to/cert.cer)")
		fmt.Fprintf(&b, "%s%-14s %s\n", pMarker, tr("証明書パス:"), f.authParam1.View())
	case vpncmd.UserAuthSignedCert:
		p1Marker := "  "
		if f.focus == userFieldAuthParam1 {
			p1Marker = "> "
		}
		f.authParam1.EchoMode = textinput.EchoNormal
		f.authParam1.Placeholder = tr("Common Name (CN: 任意)")
		fmt.Fprintf(&b, "%s%-14s %s\n", p1Marker, tr("CN (任意):"), f.authParam1.View())

		p2Marker := "  "
		if f.focus == userFieldAuthParam2 {
			p2Marker = "> "
		}
		f.authParam2.Placeholder = tr("Serial Number (任意)")
		fmt.Fprintf(&b, "%s%-14s %s\n", p2Marker, tr("Serial (任意):"), f.authParam2.View())
	case vpncmd.UserAuthNTLM:
		pMarker := "  "
		if f.focus == userFieldAuthParam1 {
			pMarker = "> "
		}
		f.authParam1.EchoMode = textinput.EchoNormal
		f.authParam1.Placeholder = tr("NTドメイン/ActiveDirectory ユーザーエイリアス (任意)")
		fmt.Fprintf(&b, "%s%-14s %s\n", pMarker, tr("ADユーザーエイリアス:"), f.authParam1.View())
	case vpncmd.UserAuthRadius:
		pMarker := "  "
		if f.focus == userFieldAuthParam1 {
			pMarker = "> "
		}
		f.authParam1.EchoMode = textinput.EchoNormal
		f.authParam1.Placeholder = tr("RADIUS User Alias (任意)")
		fmt.Fprintf(&b, "%s%-14s %s\n", pMarker, tr("RADIUSエイリアス:"), f.authParam1.View())
	}

	if f.dropdownActive {
		b.WriteString("\n")
		b.WriteString(titleStyle.Render(tr("--- 選択メニュー (↑/↓:移動 Enter:決定 Esc:閉じる) ---")) + "\n")
		if f.dropdownIsAuth {
			for i, a := range userAuthOrder {
				cur := "  "
				if i == f.dropdownCursor {
					cur = "> "
				}
				b.WriteString(cur + userAuthLabels[a] + "\n")
			}
		} else {
			items := append([]string{"(なし)"}, f.groups...)
			for i, item := range items {
				cur := "  "
				if i == f.dropdownCursor {
					cur = "> "
				}
				b.WriteString(cur + item + "\n")
			}
		}
	}

	nameValid := strings.TrimSpace(f.inputs[userFieldName].Value()) != ""
	pwValid := f.authType != vpncmd.UserAuthPassword || strings.TrimSpace(f.authParam1.Value()) != ""
	certValid := f.authType != vpncmd.UserAuthCert || strings.TrimSpace(f.authParam1.Value()) != ""
	canSave := nameValid && pwValid && certValid

	b.WriteString("\n")
	if canSave {
		b.WriteString(saveKeyStyle.Render(" [ Save ] ") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Enter", tr("選択/作成"), "←→", tr("切替"), "Esc", tr("キャンセル")))
	} else {
		b.WriteString(dimStyle.Render("  [ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Enter", tr("選択"), "←→", tr("切替"), "Esc", tr("キャンセル")))
	}
	return b.String()
}
