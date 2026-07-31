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

type editableGroupField int

const (
	fieldGroupRealName editableGroupField = iota
	fieldGroupNote
	editableGroupFieldCount
)

type groupDetailState struct {
	profile   config.Profile
	hubName   string
	groupName string
	info      vpncmd.KeyValue
	loading   bool
	err       error

	cursor       editableGroupField
	editing      bool
	editingField editableGroupField
	input        textinput.Model
	editedValues map[editableGroupField]string
	dirty        bool
}

type groupDetailLoadedMsg struct {
	groupName string
	info      vpncmd.KeyValue
	err       error
}

func (m Model) fetchGroupDetail(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		info, err := client.GroupGet(ctx, target, name)
		return groupDetailLoadedMsg{groupName: name, info: info, err: err}
	}
}

func (d groupDetailState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(fmt.Sprintf(tr("グループ詳細: %s (Hub: %s)"), d.groupName, d.hubName)))
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

func (d groupDetailState) renderSections(b *strings.Builder) {
	// Section 1: Editable metadata
	d.renderFieldLine(b, "Group Name", d.getKV("Group Name", "Name"))
	d.renderEditableField(b, fieldGroupRealName, "Full Name", d.getKV("Full Name", "Group Full Name"))
	d.renderEditableField(b, fieldGroupNote, "Description", d.getKV("Description", "Group Description"))

	b.WriteString("\n")

	// Section 2: Statistics & remaining fields
	handledKeys := map[string]bool{
		"Group Name": true, "Name": true,
		"Full Name": true, "Group Full Name": true,
		"Description": true, "Group Description": true,
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

func (d groupDetailState) getKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.info[k]; ok {
			return v
		}
	}
	return ""
}

func (d groupDetailState) renderFieldLine(b *strings.Builder, label, val string) {
	if val == "" {
		val = "(None)"
	}
	fmt.Fprintf(b, "  %-28s %s\n", label+":", statusBarStyle.Render(val))
}

func (d groupDetailState) renderEditableField(b *strings.Builder, field editableGroupField, label, val string) {
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

func (m Model) handleGroupDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.groupDetail

	if d.editing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.input.Value())
			if d.editedValues == nil {
				d.editedValues = make(map[editableGroupField]string)
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
			d.editedValues = make(map[editableGroupField]string)
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
		if d.cursor < editableGroupFieldCount-1 {
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
			ti.SetValue(val)
		}
		ti.Focus()
		d.input = ti
		return m, nil

	case "s", "S":
		if d.dirty {
			return m.saveGroupDetailChanges()
		}

	case "d":
		m.confirm.Show(confirmDeleteGroup, d.groupName, fmt.Sprintf(tr("グループ %q を削除しますか?"), d.groupName))
	}
	return m, nil
}

func (d groupDetailState) getFieldValue(field editableGroupField) string {
	switch field {
	case fieldGroupRealName:
		return d.getKV("Full Name", "Group Full Name")
	case fieldGroupNote:
		return d.getKV("Description", "Group Description")
	}
	return ""
}

func (m Model) saveGroupDetailChanges() (tea.Model, tea.Cmd) {
	d := &m.groupDetail
	p := d.profile
	hub := d.hubName
	name := d.groupName

	realName := d.editedValues[fieldGroupRealName]
	note := d.editedValues[fieldGroupNote]

	opts := vpncmd.GroupSetOptions{
		RealName: realName,
		Note:     note,
	}

	d.dirty = false
	d.editedValues = make(map[editableGroupField]string)
	m.status = fmt.Sprintf(tr("グループ %q の変更を保存しています..."), name)
	m.statusErr = false

	return m, tea.Batch(
		m.setGroupSet(p, hub, name, opts),
		m.fetchGroupDetail(p, hub, name),
	)
}

func (m Model) setGroupSet(p config.Profile, hub, name string, opts vpncmd.GroupSetOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.GroupSet(ctx, target, name, opts)
		return groupActionResultMsg{action: tr("グループ設定変更"), name: name, err: err}
	}
}

type groupActionResultMsg struct {
	action string
	name   string
	err    error
}
