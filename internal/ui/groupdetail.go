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
	profile            config.Profile
	hubName            string
	groupName          string
	info               vpncmd.KeyValue
	members            []string
	allUsers           []string
	pendingMemberEdits map[string]bool
	loading            bool
	err                error

	cursor       int // 0..2 for metadata, 3.. for user rows, or buttons at the end when dirty
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
	allUsers  []string
	err       error
}

func (m Model) fetchGroupDetail(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := client.GroupGet(ctx, target, name)
		var userNames []string
		if uTable, uErr := client.UserList(ctx, target); uErr == nil {
			for _, r := range uTable.Rows {
				if uname := uTable.NameOf(r); uname != "" {
					userNames = append(userNames, uname)
				}
			}
		}
		return groupDetailLoadedMsg{groupName: name, info: res.Info, members: res.Members, allUsers: userNames, err: err}
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
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "s", tr("保存 (Save)"), "n", tr("変更を破棄 (Cancel)")))
	} else {
		b.WriteString("\n" + renderHelp("↑/↓", tr("移動"), "Space/Enter", tr("所属切り替え(トグル)"), "a", tr("手動ユーザー追加"), "d", tr("削除"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d groupDetailState) isMember(user string) bool {
	if ped, ok := d.pendingMemberEdits[user]; ok {
		return ped
	}
	for _, m := range d.members {
		if m == user {
			return true
		}
	}
	return false
}

func (d groupDetailState) isMemberDirty(user string) bool {
	if ped, ok := d.pendingMemberEdits[user]; ok {
		orig := false
		for _, m := range d.members {
			if m == user {
				orig = true
				break
			}
		}
		return ped != orig
	}
	return false
}

func (d groupDetailState) displayUsers() []string {
	if len(d.allUsers) > 0 {
		return d.allUsers
	}
	return d.members
}

func (d groupDetailState) renderSections(b *strings.Builder) {
	// Section 1: Editable metadata
	d.renderFieldLine(b, "Group Name", d.getKV("Group Name", "Name"), 0)
	d.renderEditableField(b, fieldGroupRealName, "Full Name", d.getKV("Full Name", "Group Full Name"), 1)
	d.renderEditableField(b, fieldGroupNote, "Description", d.getKV("Description", "Group Description"), 2)

	b.WriteString("\n")

	// Section 2: Hub Users (Assigned members marked [x], others marked [ ])
	users := d.displayUsers()
	assignedCount := 0
	for _, u := range users {
		if d.isMember(u) {
			assignedCount++
		}
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf(tr("Group Members (%d assigned / %d total users)"), assignedCount, len(users))) + "\n")
	if len(users) == 0 {
		b.WriteString(dimStyle.Render("  "+tr("(ユーザーが登録されていません)")) + "\n")
	} else {
		for i, user := range users {
			rowIdx := 3 + i
			marker := "  "
			style := statusBarStyle
			if d.cursor == rowIdx {
				marker = "> "
				style = selectedStyle
			}

			box := "[ ]"
			if d.isMember(user) {
				box = "[x]"
			}

			disp := box + " " + style.Render(user)
			if d.isMemberDirty(user) {
				disp += " " + selectedStyle.Render(tr("(変更あり)"))
			}
			fmt.Fprintf(b, "%s%s\n", marker, disp)
		}
	}

	// Action buttons when dirty
	if d.dirty {
		b.WriteString("\n")
		saveMarker := "  "
		saveStyle := inactiveTabStyle
		if d.cursor == 3+len(users) {
			saveMarker = "> "
			saveStyle = saveKeyStyle
		}
		saveText := " [ " + tr("保存 (Save)") + " ] "

		cancelMarker := "  "
		cancelStyle := inactiveTabStyle
		if d.cursor == 4+len(users) {
			cancelMarker = "> "
			cancelStyle = saveKeyStyle
		}
		cancelText := " [ " + tr("キャンセル (Cancel)") + " ] "

		b.WriteString(saveMarker + saveStyle.Render(saveText) + "  " + cancelMarker + cancelStyle.Render(cancelText) + "\n")
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

	users := d.displayUsers()
	maxCursor := 2 + len(users)
	if d.dirty {
		maxCursor = 4 + len(users)
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

	case "a", "A":
		if !d.dirty {
			m.prompt.Show(promptAddGroupMember, d.groupName, fmt.Sprintf(tr("グループ %q に追加するユーザー名:"), d.groupName), tr("ユーザー名"), false)
			return m, nil
		}

	case "c", "C":
		if !d.dirty {
			m.prompt.Show(promptAddGroupMember, d.groupName, fmt.Sprintf(tr("グループ %q に追加するユーザー名:"), d.groupName), tr("ユーザー名"), false)
			return m, nil
		}

	case "n", "N":
		if d.dirty {
			m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
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
		if d.dirty && d.cursor >= 3+len(users) {
			if d.cursor == 3+len(users) {
				d.cursor = 4 + len(users)
			} else {
				d.cursor = 3 + len(users)
			}
			return m, nil
		}

	case " ", "enter":
		// Action buttons when dirty
		if d.dirty && d.cursor >= 3+len(users) {
			if d.cursor == 3+len(users) {
				return m.saveGroupDetailChanges()
			}
			m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
			return m, nil
		}

		// Stage group membership toggle if cursor is on a user row (cursor >= 3)
		if d.cursor >= 3 && d.cursor < 3+len(users) {
			userIdx := d.cursor - 3
			userName := users[userIdx]
			isMem := d.isMember(userName)
			if d.pendingMemberEdits == nil {
				d.pendingMemberEdits = make(map[string]bool)
			}
			d.pendingMemberEdits[userName] = !isMem
			d.dirty = true
			return m, nil
		}

		if msg.String() == "enter" && (d.cursor == 1 || d.cursor == 2) {
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

	pendingEdits := make(map[string]bool)
	for u, ped := range d.pendingMemberEdits {
		orig := false
		for _, mem := range d.members {
			if mem == u {
				orig = true
				break
			}
		}
		if ped != orig {
			pendingEdits[u] = ped
		}
	}

	d.dirty = false
	d.editedValues = make(map[editableGroupField]string)
	d.pendingMemberEdits = make(map[string]bool)
	m.status = fmt.Sprintf(tr("グループ %q の変更を保存しています..."), name)
	m.statusErr = false

	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if realName != "" || note != "" {
			_ = client.GroupSet(ctx, target, name, opts)
		}
		for u, assign := range pendingEdits {
			g := name
			if !assign {
				g = ""
			}
			_ = client.UserSet(ctx, target, u, vpncmd.UserSetOptions{Group: g})
		}

		res, err := client.GroupGet(ctx, target, name)
		var userNames []string
		if uTable, uErr := client.UserList(ctx, target); uErr == nil {
			for _, r := range uTable.Rows {
				if uname := uTable.NameOf(r); uname != "" {
					userNames = append(userNames, uname)
				}
			}
		}
		return groupDetailLoadedMsg{groupName: name, info: res.Info, members: res.Members, allUsers: userNames, err: err}
	}
}
