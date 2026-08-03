package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

type editableUserField int

const (
	fieldRealName editableUserField = iota
	fieldGroup
	fieldNote
	fieldAuthType
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

	// Auth Type Dropdown overlay state
	dropdownActive bool
	dropdownCursor int
	authType       vpncmd.UserAuthType
	authParam1     string
	authParam2     string

	// Group Dropdown overlay state
	groups              []string
	groupDropdownActive bool
	groupDropdownCursor int

	// Radius Server Info (for previewing Hub's RADIUS server settings)
	radiusServer string
}

type userDetailLoadedMsg struct {
	userName     string
	info         vpncmd.KeyValue
	radiusServer string
	err          error
}

func (m Model) fetchUserDetail(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		info, err := client.UserGet(ctx, target, name)
		radInfo, _ := client.RadiusServerGet(ctx, target)
		radServer := ""
		if radInfo != nil {
			if s, ok := radInfo["RADIUS Server Name"]; ok && s != "" {
				radServer = s
			}
		}
		return userDetailLoadedMsg{userName: name, info: info, radiusServer: radServer, err: err}
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

	if d.groupDropdownActive {
		b.WriteString("\n" + d.renderGroupDropdown())
	} else if d.dropdownActive {
		b.WriteString("\n" + d.renderAuthTypeDropdown())
	} else if d.editing {
		b.WriteString("\n" + renderHelp("Enter", tr("決定"), "Esc", tr("キャンセル")))
	} else if d.dirty {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "s", tr("保存 (Save)"), "n", tr("変更を破棄 (Cancel)")))
	} else {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "d", tr("削除"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d userDetailState) renderGroupDropdown() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr("--- 所属グループの選択 (↑/↓:移動 Enter:決定 Esc:閉じる) ---")) + "\n")
	options := append([]string{tr("(なし / 所属解除)")}, d.groups...)

	for i, opt := range options {
		marker := "  "
		style := statusBarStyle
		if d.groupDropdownCursor == i {
			marker = "> "
			style = selectedStyle
		}
		fmt.Fprintf(&b, "%s%s\n", marker, style.Render(opt))
	}
	return b.String()
}

func (d userDetailState) renderAuthTypeDropdown() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr("--- 認証方式の選択 (↑/↓:移動 Enter:決定 Esc:閉じる) ---")) + "\n")
	options := []struct {
		authType vpncmd.UserAuthType
		label    string
	}{
		{vpncmd.UserAuthPassword, "Password Authentication (パスワード認証)"},
		{vpncmd.UserAuthAnonymous, "Anonymous Authentication (匿名認証)"},
		{vpncmd.UserAuthRadius, "RADIUS Authentication (RADIUS認証)"},
		{vpncmd.UserAuthCert, "Individual Certificate Authentication (個別証明書認証)"},
		{vpncmd.UserAuthSignedCert, "Signed Certificate Authentication (CA署名付き証明書認証)"},
		{vpncmd.UserAuthNTLM, "NT Domain / Active Directory Authentication (NTドメイン/AD認証)"},
	}

	for i, opt := range options {
		marker := "  "
		style := statusBarStyle
		if d.dropdownCursor == i {
			marker = "> "
			style = selectedStyle
		}
		fmt.Fprintf(&b, "%s%s\n", marker, style.Render(opt.label))
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
	d.renderEditableField(b, fieldAuthType, "Auth Type", d.getKV("Auth Type", "Auth Method"))
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

func (d userDetailState) effectiveAuthType() vpncmd.UserAuthType {
	if d.authType != vpncmd.UserAuthNone {
		return d.authType
	}
	currentStr := d.getKV("Auth Type", "Auth Method")
	currentLower := strings.ToLower(currentStr)
	switch {
	case strings.Contains(currentLower, "radius"):
		return vpncmd.UserAuthRadius
	case strings.Contains(currentLower, "signed"):
		return vpncmd.UserAuthSignedCert
	case strings.Contains(currentLower, "cert") || strings.Contains(currentLower, "certificate"):
		return vpncmd.UserAuthCert
	case strings.Contains(currentLower, "ntlm") || strings.Contains(currentLower, "nt domain") || strings.Contains(currentLower, "active directory"):
		return vpncmd.UserAuthNTLM
	case strings.Contains(currentLower, "anonymous"):
		return vpncmd.UserAuthAnonymous
	default:
		return vpncmd.UserAuthPassword
	}
}

func (d userDetailState) renderPasswordField(b *strings.Builder) {
	auth := d.effectiveAuthType()
	if auth == vpncmd.UserAuthAnonymous {
		return
	}

	label := tr("Password (reset)")
	val := "********"

	switch auth {
	case vpncmd.UserAuthRadius:
		label = tr("RADIUS User Alias")
		val = d.authParam1
		if val == "" {
			val = tr("(None / Same as User Name)")
		}
		radSrv := d.radiusServer
		if radSrv == "" {
			radSrv = tr("(Not Configured - Press R in Overview)")
		}
		val += fmt.Sprintf("  [%s: %s]", tr("RADIUS Server"), radSrv)
	case vpncmd.UserAuthCert:
		label = tr("Certificate File Path")
		val = d.authParam1
		if val == "" {
			val = tr("(None)")
		}
	case vpncmd.UserAuthSignedCert:
		label = tr("Signed Cert CN/Serial")
		val = d.authParam1
		if d.authParam2 != "" {
			val += " / Serial: " + d.authParam2
		}
		if val == "" {
			val = tr("(None)")
		}
	case vpncmd.UserAuthNTLM:
		label = tr("NT Domain User Alias")
		val = d.authParam1
		if val == "" {
			val = tr("(None / Same as User Name)")
		}
	default:
		if ed, ok := d.editedValues[fieldPassword]; ok && ed != "" {
			val = ed + " " + selectedStyle.Render(tr("(変更あり)"))
		}
	}

	if (auth == vpncmd.UserAuthRadius || auth == vpncmd.UserAuthCert || auth == vpncmd.UserAuthSignedCert || auth == vpncmd.UserAuthNTLM) && d.authParam1 != "" {
		val += " " + selectedStyle.Render(tr("(変更あり)"))
	}

	marker := "  "
	style := statusBarStyle
	if d.cursor == fieldPassword {
		marker = "> "
		style = selectedStyle
	}

	if d.editing && d.editingField == fieldPassword {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", d.input.View())
	} else {
		fmt.Fprintf(b, "%s%-28s %s\n", marker, label+":", style.Render(val))
	}
}

func (m Model) handleUserDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.userDetail

	if d.groupDropdownActive {
		switch msg.String() {
		case "up", "k":
			if d.groupDropdownCursor > 0 {
				d.groupDropdownCursor--
			}
			return m, nil
		case "down", "j":
			if d.groupDropdownCursor < len(d.groups) {
				d.groupDropdownCursor++
			}
			return m, nil
		case "enter":
			val := ""
			if d.groupDropdownCursor > 0 && d.groupDropdownCursor-1 < len(d.groups) {
				val = d.groups[d.groupDropdownCursor-1]
			}
			if d.editedValues == nil {
				d.editedValues = make(map[editableUserField]string)
			}
			d.editedValues[fieldGroup] = val
			d.dirty = true
			d.groupDropdownActive = false
			return m, nil
		case "esc":
			d.groupDropdownActive = false
			return m, nil
		}
		return m, nil
	}

	if d.dropdownActive {
		switch msg.String() {
		case "up", "k":
			if d.dropdownCursor > 0 {
				d.dropdownCursor--
			}
			return m, nil
		case "down", "j":
			if d.dropdownCursor < 5 {
				d.dropdownCursor++
			}
			return m, nil
		case "enter":
			types := []vpncmd.UserAuthType{
				vpncmd.UserAuthPassword,
				vpncmd.UserAuthAnonymous,
				vpncmd.UserAuthRadius,
				vpncmd.UserAuthCert,
				vpncmd.UserAuthSignedCert,
				vpncmd.UserAuthNTLM,
			}
			labels := []string{
				"Password Authentication",
				"Anonymous Authentication",
				"RADIUS Authentication",
				"Individual Certificate Authentication",
				"Signed Certificate Authentication",
				"NT Domain / Active Directory Authentication",
			}
			selectedAuth := types[d.dropdownCursor]
			d.authType = selectedAuth
			if d.editedValues == nil {
				d.editedValues = make(map[editableUserField]string)
			}
			d.editedValues[fieldAuthType] = labels[d.dropdownCursor]
			d.dirty = true
			d.dropdownActive = false

			switch selectedAuth {
			case vpncmd.UserAuthRadius:
				m.prompt.Show(promptUserRadiusAlias, d.userName, tr("RADIUS User Alias (任意, 空欄でユーザー名と同等):"), d.authParam1, false)
			case vpncmd.UserAuthCert:
				m.prompt.Show(promptUserCertPath, d.userName, tr("証明書ファイルパス (/path/to/cert.cer):"), d.authParam1, false)
			case vpncmd.UserAuthNTLM:
				m.prompt.Show(promptUserNTLMAlias, d.userName, tr("NT Domain / AD User Alias (任意, 空欄でユーザー名と同等):"), d.authParam1, false)
			case vpncmd.UserAuthSignedCert:
				m.prompt.Show(promptUserSignedCN, d.userName, tr("CA署名付き証明書の許容 Common Name (CN, 任意):"), d.authParam1, false)
			}
			return m, nil
		case "esc":
			d.dropdownActive = false
			return m, nil
		}
		return m, nil
	}

	if d.editing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.input.Value())
			if d.editingField == fieldExpires && val != "" {
				// Normalize and auto-format date to YYYY/MM/DD
				val = normalizeAndFormatDate(val)
				if val == "" {
					m.status = tr("有効期限は YYYY/MM/DD 形式（数字8桁等）で入力してください")
					m.statusErr = true
					return m, nil
				}
			}
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

		if d.editingField == fieldExpires {
			// Restrict Expiration Date input to numbers only (and / or backspace/delete)
			key := msg.String()
			if len(key) == 1 && (key < "0" || key > "9") && key != "/" {
				return m, nil
			}
		}

		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd
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
		if d.cursor < editableUserFieldCount-1 {
			d.cursor++
		}

	case "enter":
		if d.cursor == fieldGroup {
			d.groupDropdownActive = true
			currentGroup := d.editedValues[fieldGroup]
			if currentGroup == "" {
				currentGroup = d.getKV("Group Name", "Group")
			}
			d.groupDropdownCursor = 0
			for i, g := range d.groups {
				if g == currentGroup {
					d.groupDropdownCursor = i + 1
					break
				}
			}
			return m, nil
		}

		if d.cursor == fieldAuthType {
			d.dropdownActive = true
			d.dropdownCursor = 0
			return m, nil
		}

		if d.cursor == fieldPassword {
			auth := d.effectiveAuthType()
			switch auth {
			case vpncmd.UserAuthRadius:
				m.prompt.Show(promptUserRadiusAlias, d.userName, tr("RADIUS User Alias (任意, 空欄でユーザー名と同等):"), d.authParam1, false)
				return m, nil
			case vpncmd.UserAuthCert:
				m.prompt.Show(promptUserCertPath, d.userName, tr("証明書ファイルパス (/path/to/cert.cer):"), d.authParam1, false)
				return m, nil
			case vpncmd.UserAuthNTLM:
				m.prompt.Show(promptUserNTLMAlias, d.userName, tr("NT Domain / AD User Alias (任意, 空欄でユーザー名と同等):"), d.authParam1, false)
				return m, nil
			case vpncmd.UserAuthSignedCert:
				m.prompt.Show(promptUserSignedCN, d.userName, tr("CA署名付き証明書の許容 Common Name (CN, 任意):"), d.authParam1, false)
				return m, nil
			case vpncmd.UserAuthAnonymous:
				return m, nil
			}
		}

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
	case fieldRealName:
		return d.getKV("Full Name", "User Full Name")
	case fieldGroup:
		return d.getKV("Group Name", "Group")
	case fieldNote:
		return d.getKV("Description", "User Description")
	case fieldAuthType:
		return d.getKV("Auth Type", "Auth Method")
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

	// Change Auth Type if updated
	if d.authType != vpncmd.UserAuthNone {
		client := m.client
		target := m.targetFromProfile(p).WithHub(hub)
		authType := d.authType
		pw := d.editedValues[fieldPassword]

		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			var err error
			switch authType {
			case vpncmd.UserAuthPassword:
				err = client.UserPasswordSet(ctx, target, name, pw)
			case vpncmd.UserAuthAnonymous:
				err = client.UserAnonymousSet(ctx, target, name)
			case vpncmd.UserAuthRadius:
				err = client.UserRadiusSet(ctx, target, name, d.authParam1)
			case vpncmd.UserAuthCert:
				err = client.UserCertSet(ctx, target, name, d.authParam1)
			case vpncmd.UserAuthSignedCert:
				err = client.UserSignedSet(ctx, target, name, d.authParam1, d.authParam2)
			case vpncmd.UserAuthNTLM:
				err = client.UserNTLMSet(ctx, target, name, d.authParam1)
			}
			return userActionResultMsg{action: tr("認証方式変更"), name: name, err: err}
		})
	} else if pw, ok := d.editedValues[fieldPassword]; ok && pw != "" {
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

// normalizeAndFormatDate parses raw input (e.g. "2026111", "2026/11/1", "20261101")
// and returns a normalized YYYY/MM/DD string (e.g. "2026/11/01"), or empty string if invalid.
func normalizeAndFormatDate(input string) string {
	clean := strings.ReplaceAll(input, "-", "/")
	parts := strings.Split(clean, "/")

	if len(parts) == 3 {
		yearStr, monthStr, dayStr := parts[0], parts[1], parts[2]
		year, yErr := strconv.Atoi(yearStr)
		month, mErr := strconv.Atoi(monthStr)
		day, dErr := strconv.Atoi(dayStr)
		if yErr == nil && mErr == nil && dErr == nil && year >= 2000 && year <= 2100 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d/%02d/%02d", year, month, day)
		}
	} else if len(parts) == 1 {
		// Pure digits e.g. "20261101" or "2026111"
		digits := parts[0]
		if len(digits) == 8 {
			year, _ := strconv.Atoi(digits[:4])
			month, _ := strconv.Atoi(digits[4:6])
			day, _ := strconv.Atoi(digits[6:8])
			if year >= 2000 && year <= 2100 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
				return fmt.Sprintf("%04d/%02d/%02d", year, month, day)
			}
		}
	}

	// Fallback attempt with time.Parse
	t, err := time.Parse("2006/1/2", clean)
	if err == nil {
		return t.Format("2006/01/02")
	}
	return ""
}
