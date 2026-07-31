package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

type editableUserField int

const (
	fieldGroup editableUserField = iota
	fieldRealName
	fieldNote
	fieldPassword
	fieldExpires
	editableUserFieldCount
)

type userDetailState struct {
	profile  config.Profile
	hubName  string
	userName string
	info     vpncmd.KeyValue
	loading  bool
	err      error

	// Interactive field navigation & editing
	cursor       editableUserField
	editing      bool
	editingField editableUserField
	input        textinput.Model
	editedValues map[editableUserField]string
	dirty        bool
}

type userDetailLoadedMsg struct {
	userName string
	info     vpncmd.KeyValue
	err      error
}

func (m Model) fetchUserDetail(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		info, err := client.UserGet(ctx, target, name)
		return userDetailLoadedMsg{userName: name, info: info, err: err}
	}
}

func (d userDetailState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(fmt.Sprintf(tr("ユーザー詳細: %s (Hub: %s)"), d.userName, d.hubName)))
	b.WriteString(strings.Repeat("─", 60) + "\n")

	switch {
	case d.loading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.err != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.err.Error()) + "\n")
	default:
		d.renderSections(&b)
	}

	if d.editing {
		b.WriteString("\n" + renderHelp("Enter", tr("決定"), "Esc", tr("キャンセル")))
	} else if d.dirty {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "s", tr("保存 (Save)"), "c", tr("変更を破棄 (Cancel)")))
	} else {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "d", tr("削除"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d userDetailState) renderSections(b *strings.Builder) {
	// Group 1: Editable user settings & account metadata
	// Ordered: User Name, Full Name, Group Name, Description, Auth Type, Password (reset), Expiration Date, Created on, Updated on
	d.renderFieldLine(b, "User Name", d.getKV("User Name", "Name"))
	d.renderEditableField(b, fieldRealName, "Full Name", d.getKV("Full Name", "User Full Name"))
	d.renderEditableField(b, fieldGroup, "Group Name", d.getKV("Group Name", "Group"))
	d.renderEditableField(b, fieldNote, "Description", d.getKV("Description", "User Description"))
	d.renderFieldLine(b, "Auth Type", d.getKV("Auth Type", "Auth Method"))
	d.renderPasswordField(b)
	d.renderEditableField(b, fieldExpires, "Expiration Date", d.getKV("Expiration Date", "Expiration Date (UTC)"))
	d.renderFieldLine(b, "Created on", d.getKV("Created on", "Created at"))
	d.renderFieldLine(b, "Updated on", d.getKV("Updated on", "Updated at"))

	b.WriteString("\n")

	// Group 2: Read-only statistics & status (separated by blank line)
	// Render remaining items sorted
	handledKeys := map[string]bool{
		"User Name": true, "Name": true,
		"Full Name": true, "User Full Name": true,
		"Group Name": true, "Group": true,
		"Description": true, "User Description": true,
		"Auth Type": true, "Auth Method": true,
		"Expiration Date": true, "Expiration Date (UTC)": true,
		"Created on": true, "Created at": true,
		"Updated on": true, "Updated at": true,
		"---": true,
	}

	keys := make([]string, 0, len(d.info))
	for k := range d.info {
		if !handledKeys[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(b, "  %-28s %s\n", k+":", statusBarStyle.Render(d.info[k]))
	}
}

func (d userDetailState) getKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.info[k]; ok {
			return v
		}
	}
	return ""
}

func (d userDetailState) renderFieldLine(b *strings.Builder, label, val string) {
	if val == "" {
		val = "(None)"
	}
	fmt.Fprintf(b, "  %-28s %s\n", label+":", statusBarStyle.Render(val))
}

func (d userDetailState) renderEditableField(b *strings.Builder, field editableUserField, label, val string) {
	if ed, ok := d.editedValues[field]; ok {
		val = ed + " " + selectedStyle.Render(tr("(変更あり)"))
	} else if val == "" {
		val = "(None)"
	}

	marker := "  "
	style := statusBarStyle
	if d.cursor == field {
		marker = "> "
		style = selectedStyle
	}

	if d.editing && d.editingField == field {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", d.input.View())
	} else {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", style.Render(val))
	}
}

func (d userDetailState) renderPasswordField(b *strings.Builder) {
	pwVal := "********"
	if ed, ok := d.editedValues[fieldPassword]; ok && ed != "" {
		pwVal = ed + " " + selectedStyle.Render(tr("(変更あり)"))
	}
	marker := "  "
	style := statusBarStyle
	if d.cursor == fieldPassword {
		marker = "> "
		style = selectedStyle
	}

	label := tr("Password (reset)")
	if d.editing && d.editingField == fieldPassword {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", d.input.View())
	} else {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", style.Render(pwVal))
	}
}

func (m Model) handleUserDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.userDetail

	if d.editing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.input.Value())
			if d.editedValues == nil {
				d.editedValues = make(map[editableUserField]string)
			}
			d.editedValues[d.editingField] = val
			d.dirty = true
			d.editing = false
			return m, nil

		case "esc":
			d.editing = false
			return m, nil
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace", "c", "C":
		if d.dirty && (msg.String() == "c" || msg.String() == "C") {
			d.editedValues = make(map[editableUserField]string)
			d.dirty = false
			m.status = tr("変更を破棄しました")
			m.statusErr = false
			return m, nil
		}
		m.screen = screenHubDetail
		return m, nil

	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}

	case "down", "j":
		if d.cursor < editableUserFieldCount-1 {
			d.cursor++
		}

	case "enter":
		d.editing = true
		d.editingField = d.cursor
		ti := textinput.New()
		if prev, ok := d.editedValues[d.cursor]; ok {
			ti.SetValue(prev)
		} else {
			val := d.getFieldValue(d.cursor)
			if val == "(None)" {
				val = ""
			}
			// If Expiration Date has full timestamp format (e.g. 2026-07-30 (Thu) 23:38:44), truncate to YYYY/MM/DD
			if d.cursor == fieldExpires && len(val) >= 10 {
				val = strings.ReplaceAll(val[:10], "-", "/")
			}
			ti.SetValue(val)
		}
		if d.cursor == fieldPassword {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '*'
		}
		ti.Focus()
		d.input = ti
		return m, nil

	case "s", "S":
		if d.dirty {
			return m.saveUserDetailChanges()
		}

	case "d":
		m.confirm.Show(confirmDeleteUser, d.userName, fmt.Sprintf(tr("ユーザー %q を削除しますか?"), d.userName))
	}
	return m, nil
}

func (d userDetailState) getFieldValue(field editableUserField) string {
	switch field {
	case fieldGroup:
		return d.getKV("Group Name", "Group")
	case fieldRealName:
		return d.getKV("Full Name", "User Full Name")
	case fieldNote:
		return d.getKV("Description", "User Description")
	case fieldExpires:
		return d.getKV("Expiration Date", "Expiration Date (UTC)")
	}
	return ""
}

func (m Model) saveUserDetailChanges() (tea.Model, tea.Cmd) {
	d := &m.userDetail
	p := d.profile
	hub := d.hubName
	name := d.userName
	var cmds []tea.Cmd

	group := d.editedValues[fieldGroup]
	realName := d.editedValues[fieldRealName]
	note := d.editedValues[fieldNote]

	// UserSet if any metadata changed
	if group != "" || realName != "" || note != "" {
		opts := vpncmd.UserSetOptions{
			Group:    group,
			RealName: realName,
			Note:     note,
		}
		cmds = append(cmds, m.setUserSet(p, hub, name, opts))
	}

	if pw, ok := d.editedValues[fieldPassword]; ok && pw != "" {
		cmds = append(cmds, m.setUserPassword(p, hub, name, pw))
	}

	if expStr, ok := d.editedValues[fieldExpires]; ok && expStr != "" {
		expClean := strings.TrimSpace(expStr)
		if expClean != "" {
			expClean = strings.ReplaceAll(expClean, "-", "/")
			expires, err := time.Parse("2006/01/02", expClean)
			if err == nil {
				cmds = append(cmds, m.setUserExpires(p, hub, name, expires))
			} else {
				m.status = tr("有効期限は YYYY/MM/DD 形式で入力してください")
				m.statusErr = true
				return m, nil
			}
		}
	}

	d.dirty = false
	d.editedValues = make(map[editableUserField]string)
	m.status = fmt.Sprintf(tr("ユーザー %q の変更を保存しています..."), name)
	m.statusErr = false

	cmds = append(cmds, m.fetchUserDetail(p, hub, name))
	return m, tea.Batch(cmds...)
}

func (m Model) setUserSet(p config.Profile, hub, name string, opts vpncmd.UserSetOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.UserSet(ctx, target, name, opts)
		return userActionResultMsg{action: tr("ユーザー設定変更"), name: name, err: err}
	}
}
