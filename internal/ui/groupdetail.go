package ui

import (
	"context"
	"fmt"
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
	profile         config.Profile
	hubName         string
	groupName       string
	info            vpncmd.KeyValue
	members         []string
	selectedMembers map[string]bool
	loading         bool
	err             error

	cursor       int // 0..1 for editable fields, 2..2+len(members)-1 for member rows, or 2+len(members) for remove button
	editing      bool
	editingField editableGroupField
	input        textinput.Model
	editedValues map[editableGroupField]string
	dirty        bool
}

type groupDetailLoadedMsg struct {
	groupName string
	info      vpncmd.KeyValue
	members   []string
	err       error
}

func (m Model) fetchGroupDetail(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := client.GroupGet(ctx, target, name)
		return groupDetailLoadedMsg{groupName: name, info: res.Info, members: res.Members, err: err}
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

	selectedCount := d.selectedCount()
	if d.editing {
		b.WriteString("\n" + renderHelp("Enter", tr("決定"), "Esc", tr("キャンセル")))
	} else if d.dirty {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "s", tr("保存 (Save)"), "c", tr("変更を破棄 (Cancel)")))
	} else if selectedCount > 0 {
		b.WriteString("\n" + renderHelp("↑/↓", tr("移動"), "Space", tr("選択解除/トグル"), "r", tr("選択ユーザーをグループ解除"), "Esc", tr("戻る")))
	} else {
		b.WriteString("\n" + renderHelp("↑/↓", tr("移動"), "Space", tr("ユーザー選択"), "Enter", tr("値の変更"), "d", tr("削除"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d groupDetailState) selectedCount() int {
	c := 0
	for _, sel := range d.selectedMembers {
		if sel {
			c++
		}
	}
	return c
}

func (d groupDetailState) renderSections(b *strings.Builder) {
	// Section 1: Editable metadata
	d.renderFieldLine(b, "Group Name", d.getKV("Group Name", "Name"), 0)
	d.renderEditableField(b, fieldGroupRealName, "Full Name", d.getKV("Full Name", "Group Full Name"), 1)
	d.renderEditableField(b, fieldGroupNote, "Description", d.getKV("Description", "Group Description"), 2)

	b.WriteString("\n")

	// Section 2: Members (Assigned users with checkboxes)
	b.WriteString(headerStyle.Render(fmt.Sprintf(tr("Members (所属ユーザー: %d名)"), len(d.members))) + "\n")
	if len(d.members) == 0 {
		b.WriteString(dimStyle.Render(tr("  (所属しているユーザーはいません)")) + "\n")
	} else {
		for i, member := range d.members {
			rowIdx := 3 + i
			marker := "  "
			style := statusBarStyle
			if d.cursor == rowIdx {
				marker = "> "
				style = selectedStyle
			}

			box := "[ ]"
			if d.selectedMembers != nil && d.selectedMembers[member] {
				box = "[x]"
			}

			fmt.Fprintf(b, "%s%s %s\n", marker, box, style.Render(member))
		}
	}

	selectedCount := d.selectedCount()
	if selectedCount > 0 {
		b.WriteString("\n")
		remMarker := "  "
		remStyle := inactiveTabStyle
		if d.cursor == 3+len(d.members) {
			remMarker = "> "
			remStyle = saveKeyStyle
		}
		remText := fmt.Sprintf(tr(" [ 選択した %d 名のユーザーをグループから解除 (Remove) ] "), selectedCount)

		cancelMarker := "  "
		cancelStyle := inactiveTabStyle
		if d.cursor == 4+len(d.members) {
			cancelMarker = "> "
			cancelStyle = saveKeyStyle
		}
		cancelText := tr(" [ 選択解除 (Cancel) ] ")

		b.WriteString(remMarker + remStyle.Render(remText) + "  " + cancelMarker + cancelStyle.Render(cancelText) + "\n")
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

func (d groupDetailState) renderFieldLine(b *strings.Builder, label, val string, cursorIdx int) {
	if val == "" {
		val = "(None)"
	}
	marker := "  "
	style := statusBarStyle
	if d.cursor == cursorIdx {
		marker = "> "
		style = selectedStyle
	}
	fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", style.Render(val))
}

func (d groupDetailState) renderEditableField(b *strings.Builder, field editableGroupField, label, val string, cursorIdx int) {
	if ed, ok := d.editedValues[field]; ok {
		val = ed + " " + selectedStyle.Render(tr("(変更あり)"))
	} else if val == "" {
		val = "(None)"
	}

	marker := "  "
	style := statusBarStyle
	if d.cursor == cursorIdx {
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

	maxCursor := 2 + len(d.members)
	if d.selectedCount() > 0 {
		maxCursor = 4 + len(d.members)
	}

	switch msg.String() {
	case "q":
		if d.dirty {
			m.confirm.Show(confirmQuitUnsaved, "", tr("未保存の変更があります。変更を破棄して終了しますか?"))
			return m, nil
		}
		m.confirm.Show(confirmQuitApp, "", tr("アプリケーションを終了しますか?"))
		return m, nil

	case "esc":
		if d.dirty {
			m.confirm.Show(confirmDiscardChanges, "", tr("未保存の変更があります。変更を破棄して戻りますか?"))
			return m, nil
		}
		m.screen = screenHubDetail
		return m, nil

	case "c", "C":
		if d.selectedCount() > 0 {
			d.selectedMembers = make(map[string]bool)
			return m, nil
		}
		if d.dirty {
			d.editedValues = make(map[editableGroupField]string)
			d.dirty = false
			m.status = tr("変更を破棄しました")
			m.statusErr = false
			return m, nil
		}

	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}

	case "down", "j":
		if d.cursor < maxCursor {
			d.cursor++
		}

	case "left", "right":
		if d.selectedCount() > 0 && d.cursor >= 3+len(d.members) {
			if d.cursor == 3+len(d.members) {
				d.cursor = 4 + len(d.members)
			} else {
				d.cursor = 3 + len(d.members)
			}
			return m, nil
		}

	case " ":
		// Toggle checkbox if on a member row (cursors 3 .. 3+len(members)-1)
		if d.cursor >= 3 && d.cursor < 3+len(d.members) {
			memberIdx := d.cursor - 3
			memberName := d.members[memberIdx]
			if d.selectedMembers == nil {
				d.selectedMembers = make(map[string]bool)
			}
			d.selectedMembers[memberName] = !d.selectedMembers[memberName]
			return m, nil
		}

	case "r", "R":
		if d.selectedCount() > 0 {
			m.confirm.Show(confirmRemoveGroupMembers, d.groupName, fmt.Sprintf(tr("選択した %d 名のユーザーをグループ %q から解除しますか?"), d.selectedCount(), d.groupName))
			return m, nil
		}

	case "enter":
		if d.selectedCount() > 0 {
			if d.cursor == 4+len(d.members) {
				d.selectedMembers = make(map[string]bool)
				d.cursor = 3
				return m, nil
			}
			if d.cursor >= 3 {
				m.confirm.Show(confirmRemoveGroupMembers, d.groupName, fmt.Sprintf(tr("選択した %d 名のユーザーをグループ %q から解除しますか?"), d.selectedCount(), d.groupName))
				return m, nil
			}
		}
		if d.cursor == 1 || d.cursor == 2 {
			d.editing = true
			if d.cursor == 1 {
				d.editingField = fieldGroupRealName
			} else {
				d.editingField = fieldGroupNote
			}
			ti := textinput.New()
			if prev, ok := d.editedValues[d.editingField]; ok {
				ti.SetValue(prev)
			} else {
				val := d.getFieldValue(d.editingField)
				if val == "(None)" {
					val = ""
				}
				ti.SetValue(val)
			}
			ti.Focus()
			d.input = ti
			return m, nil
		}

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
