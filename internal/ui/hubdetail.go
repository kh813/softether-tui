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

type hubTab int

const (
	hubTabOverview hubTab = iota
	hubTabSessions
	hubTabUsersAndGroups
	hubTabSecureNAT
	hubTabACL
	hubTabCascade
	hubTabLog
	hubTabCount
)

func hubTabLabels() [hubTabCount]string {
	return [hubTabCount]string{
		tr("概要"), tr("セッション"), tr("Users & Group"), "SecureNAT", "ACL", tr("カスケード"), tr("セキュリティログ"),
	}
}

// hubDetailState is the Hub detail screen.
type hubDetailState struct {
	profile config.Profile
	hubName string
	tab     hubTab

	info    vpncmd.KeyValue
	loading bool
	err     error

	users             vpncmd.Table
	usersLoaded       bool
	usersLoading      bool
	usersErr          error
	userCursor        int
	userFilter        string
	filtering         bool
	filterInput       textinput.Model
	activeUserSection bool // false = Users section, true = Groups section

	groups        vpncmd.Table
	groupsLoaded  bool
	groupsLoading bool
	groupsErr     error
	groupCursor   int

	sessions        vpncmd.Table
	sessionsLoaded  bool
	sessionsLoading bool
	sessionsErr     error
	sessionCursor   int
	refreshInterval time.Duration
	lastRefreshed   time.Time
	sessionGen      int // invalidates stale auto-refresh tick chains

	logInfo    vpncmd.KeyValue
	logLoaded  bool
	logLoading bool
	logErr     error
	logCursor  int

	secureNatStatus       vpncmd.KeyValue
	secureNatHubStatus    vpncmd.KeyValue
	secureNatHost         vpncmd.KeyValue
	secureNatDhcp         vpncmd.KeyValue
	secureNatLoaded       bool
	secureNatLoading      bool
	secureNatErr          error
	secureNatCursor       editableSecureNATField
	secureNatEditing      bool
	secureNatEditingField editableSecureNATField
	secureNatEditedValues map[editableSecureNATField]string
	secureNatDirty        bool

	access        vpncmd.Table
	accessLoaded  bool
	accessLoading bool
	accessErr     error
	accessCursor  int

	cascade        vpncmd.Table
	cascadeLoaded  bool
	cascadeLoading bool
	cascadeErr     error
	cascadeCursor  int
}

func (d hubDetailState) filteredUsers() []vpncmd.KeyValue {
	if d.userFilter == "" {
		return d.users.Rows
	}
	needle := strings.ToLower(d.userFilter)
	var out []vpncmd.KeyValue
	for _, row := range d.users.Rows {
		for _, v := range row {
			if strings.Contains(strings.ToLower(v), needle) {
				out = append(out, row)
				break
			}
		}
	}
	return out
}

func (d hubDetailState) currentUserName() (string, bool) {
	rows := d.filteredUsers()
	if d.userCursor < 0 || d.userCursor >= len(rows) {
		return "", false
	}
	return d.users.NameOf(rows[d.userCursor]), true
}

func (d hubDetailState) currentGroupName() (string, bool) {
	if d.groupCursor < 0 || d.groupCursor >= len(d.groups.Rows) {
		return "", false
	}
	return d.groups.NameOf(d.groups.Rows[d.groupCursor]), true
}

func (d hubDetailState) currentSessionName() (string, bool) {
	if d.sessionCursor < 0 || d.sessionCursor >= len(d.sessions.Rows) {
		return "", false
	}
	return d.sessions.NameOf(d.sessions.Rows[d.sessionCursor]), true
}

func (d hubDetailState) currentAccessID() (string, bool) {
	if d.accessCursor < 0 || d.accessCursor >= len(d.access.Rows) {
		return "", false
	}
	return d.access.NameOf(d.access.Rows[d.accessCursor]), true
}

func (d hubDetailState) currentCascadeName() (string, bool) {
	if d.cascadeCursor < 0 || d.cascadeCursor >= len(d.cascade.Rows) {
		return "", false
	}
	return d.cascade.NameOf(d.cascade.Rows[d.cascadeCursor]), true
}

func (d hubDetailState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render("Hub: "+d.hubName))
	b.WriteString(renderHubTabs(d.tab) + "\n")
	b.WriteString(strings.Repeat("─", 60) + "\n")

	switch d.tab {
	case hubTabOverview:
		d.viewOverview(&b)
	case hubTabSessions:
		d.viewSessions(&b)
	case hubTabUsersAndGroups:
		d.viewUsersAndGroups(&b)
	case hubTabSecureNAT:
		d.viewSecureNAT(&b)
	case hubTabACL:
		d.viewAccessList(&b)
	case hubTabCascade:
		d.viewCascade(&b)
	case hubTabLog:
		d.viewLog(&b)
	default:
		b.WriteString(dimStyle.Render(tr("この画面は今後のマイルストーンで実装予定です。")) + "\n")
		b.WriteString("\n" + dimStyle.Render(tr("Tab/Shift+Tab:タブ切替  Esc:戻る  q:終了")))
	}
	return b.String()
}

func (d hubDetailState) viewOverview(b *strings.Builder) {
	switch {
	case d.loading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.err != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.err.Error()) + "\n")
	default:
		writeKV(b, d.info)
	}
	b.WriteString("\n" + renderHelp(
		"R", tr("RADIUS設定"),
		"o", tr("オンライン化"),
		"f", tr("オフライン化"),
	))
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

func (d hubDetailState) viewUsersAndGroups(b *strings.Builder) {
	if d.filtering {
		fmt.Fprintf(b, tr("検索: %s\n\n"), d.filterInput.View())
	} else if d.userFilter != "" {
		fmt.Fprintf(b, tr("検索: %s\n\n"), d.userFilter)
	}

	// Users Section
	b.WriteString(headerStyle.Render(tr("Users (ユーザー一覧)")) + "\n")
	switch {
	case d.usersLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.usersErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.usersErr.Error()) + "\n")
	default:
		rows := d.filteredUsers()
		userCur := -1
		if !d.activeUserSection {
			userCur = d.userCursor
		}
		b.WriteString(renderTable(vpncmd.Table{Headers: d.users.Headers, Rows: rows}, userCur))
		fmt.Fprintf(b, tr("%d件 / 全%d件\n"), len(rows), len(d.users.Rows))
	}

	b.WriteString("\n")

	// Groups Section
	b.WriteString(headerStyle.Render(tr("Groups (グループ一覧)")) + "\n")
	switch {
	case d.groupsLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.groupsErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.groupsErr.Error()) + "\n")
	default:
		groupCur := -1
		if d.activeUserSection {
			groupCur = d.groupCursor
		}
		b.WriteString(renderTable(d.groups, groupCur))
		fmt.Fprintf(b, tr("%d件 / 全%d件\n"), len(d.groups.Rows), len(d.groups.Rows))
	}

	b.WriteString("\n")
	if !d.activeUserSection {
		b.WriteString(renderHelp(
			"↑/↓", tr("選択"),
			"Enter", tr("詳細"),
			"/", tr("検索"),
			"c", tr("作成"),
			"d", tr("削除"),
			"p", tr("パスワード再設定"),
			"e", tr("有効期限設定"),
			"u/g", tr("Users/Groups切替"),
		))
	} else {
		b.WriteString(renderHelp(
			"↑/↓", tr("選択"),
			"Enter", tr("詳細"),
			"c", tr("作成"),
			"d", tr("削除"),
			"u/g", tr("Users/Groups切替"),
		))
	}
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

func (d hubDetailState) viewSessions(b *strings.Builder) {
	fmt.Fprintf(b, tr("自動更新: %s毎"), d.refreshInterval)
	if !d.lastRefreshed.IsZero() {
		fmt.Fprintf(b, tr("  最終更新: %s"), d.lastRefreshed.Format("15:04:05"))
	}
	b.WriteString("\n\n")

	switch {
	case d.sessionsLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.sessionsErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.sessionsErr.Error()) + "\n")
	default:
		b.WriteString(renderTable(d.sessions, d.sessionCursor))
	}

	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("選択"),
		"x", tr("切断"),
		"+/-", tr("更新間隔変更"),
		"r", tr("手動更新"),
	))
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

var logSettingKeys = []struct {
	key   string
	label string
}{
	{"Save Security Log", "Save Security Log"},
	{"Security Switch Cycle", "Security Log Switch Cycle"},
	{"Save Packet Log", "Save Packet Log"},
	{"Packet Switch Cycle", "Packet Log Switch Cycle"},
	{"TCP Connection Log", "TCP Connection Log (tcpconn)"},
	{"TCP Packet Log", "TCP Packet Log (tcpdata)"},
	{"DHCP Log", "DHCP Log (dhcp)"},
	{"UDP Log", "UDP Log (udp)"},
	{"ICMP Log", "ICMP Log (icmp)"},
	{"IP Log", "IP Log (ip)"},
	{"ARP Log", "ARP Log (arp)"},
	{"Ethernet Log", "Ethernet Log (ether)"},
}

func (d hubDetailState) viewLog(b *strings.Builder) {
	switch {
	case d.logLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.logErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.logErr.Error()) + "\n")
	default:
		d.renderLogFields(b)
	}
	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("項目選択"),
		"Space/Enter", tr("設定切替"),
	))
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

func (d hubDetailState) renderLogFields(b *strings.Builder) {
	b.WriteString(headerStyle.Render(tr("Log Settings (ログ保存設定)")) + "\n")
	for i, item := range logSettingKeys {
		val := d.getLogKV(item.key)
		if val == "" {
			val = "(None)"
		}
		marker := "  "
		style := statusBarStyle
		if d.logCursor == i {
			marker = "> "
			style = selectedStyle
		}
		fmt.Fprintf(b, "%s%-32s %s\n", marker, item.label+":", style.Render(val))
	}
}

func (d hubDetailState) getLogKV(key string) string {
	if val, ok := d.logInfo[key]; ok {
		return val
	}
	if key == "Security Switch Cycle" {
		return d.logInfo["Log File Switch Cycle"]
	}
	return ""
}

func (d hubDetailState) viewSecureNAT(b *strings.Builder) {
	if d.secureNatLoading {
		b.WriteString(tr("読み込み中...") + "\n")
	} else {
		d.renderSecureNATFields(b)
	}

	if d.secureNatEditing {
		b.WriteString("\n" + renderHelp("Enter", tr("決定"), "Esc", tr("キャンセル")))
	} else if d.secureNatDirty {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更/切替"), "s", tr("保存 (Save)"), "c", tr("変更を破棄 (Cancel)")))
	} else {
		b.WriteString("\n" + renderHelp(
			"↑/↓", tr("項目選択"),
			"Enter", tr("有効/無効切替・値変更"),
		))
		b.WriteString("\n" + renderHelp(
			"←/→/Tab", tr("タブ切替"),
			"r", tr("更新"),
			"Esc", tr("戻る"),
			"q", tr("終了"),
		))
	}
}

func (d hubDetailState) isSecureNatEnabled() bool {
	if v, ok := d.secureNatHubStatus["SecureNAT"]; ok {
		vLower := strings.ToLower(strings.TrimSpace(v))
		return strings.Contains(vLower, "enabled") || strings.Contains(vLower, "active") || strings.Contains(vLower, "yes") || strings.Contains(vLower, "true") || strings.Contains(vLower, "有効")
	}
	if d.secureNatErr != nil || len(d.secureNatStatus) == 0 {
		return false
	}
	for k, v := range d.secureNatStatus {
		kLower := strings.ToLower(k)
		vLower := strings.ToLower(v)
		if strings.Contains(kLower, "status") || strings.Contains(kLower, "active") || strings.Contains(kLower, "state") {
			if strings.Contains(vLower, "disabled") || strings.Contains(vLower, "stopped") || strings.Contains(vLower, "no") || strings.Contains(vLower, "false") || strings.Contains(vLower, "無効") {
				return false
			}
		}
	}
	return true
}

func (d hubDetailState) renderSecureNATFields(b *strings.Builder) {
	b.WriteString(headerStyle.Render(tr("Virtual host settings (仮想ホスト・DHCP設定)")) + "\n")

	// Render SecureNAT enabled status as a selectable field
	snEnabled := d.isSecureNatEnabled()
	snStr := ""
	if snEnabled {
		snStr = selectedStyle.Render("[enabled]")
	} else {
		snStr = selectedStyle.Render("[disabled]")
	}
	marker := "  "
	if d.secureNatCursor == fieldSecureNAT {
		marker = "> "
	}
	fmt.Fprintf(b, "%s%-32s %s\n", marker, tr("SecureNAT")+":", snStr)

	d.renderEditableNatField(b, fieldNatIP, "IP Address", d.getNatHostKV("IP Address", "IP"))
	d.renderEditableNatField(b, fieldNatMask, "Subnet Mask", d.getNatHostKV("Subnet Mask", "Mask"))
	d.renderEditableNatField(b, fieldNatMAC, "MAC Address", d.getNatHostKV("MAC Address", "MAC"))
	d.renderEditableNatField(b, fieldNatMTU, "MTU", d.getNatHostKV("MTU", "Mtu"))

	// Render DHCP enabled status as a selectable field
	dhcpEnabled := false
	for _, k := range []string{"Use Virtual DHCP Function", "Use Virtual DHCP Server", "Virtual DHCP Server", "Use DHCP", "DHCP Server", "Status"} {
		if v, ok := d.secureNatDhcp[k]; ok {
			vLower := strings.ToLower(v)
			if strings.Contains(vLower, "yes") || strings.Contains(vLower, "enable") || strings.Contains(vLower, "active") || strings.Contains(vLower, "true") {
				dhcpEnabled = true
				break
			}
		}
	}
	dhcpStr := ""
	if dhcpEnabled {
		dhcpStr = selectedStyle.Render("[enabled]")
	} else {
		dhcpStr = selectedStyle.Render("[disabled]")
	}
	marker = "  "
	if d.secureNatCursor == fieldDHCP {
		marker = "> "
	}
	fmt.Fprintf(b, "%s%-32s %s\n", marker, tr("DHCP")+":", dhcpStr)

	startIp := d.getNatDhcpKV("Start Distribution Address Band", "Start")
	endIp := d.getNatDhcpKV("End Distribution Address Band", "End")
	rangeVal := startIp
	if endIp != "" {
		rangeVal = startIp + " - " + endIp
	}
	d.renderEditableNatField(b, fieldDhcpRange, "DHCP Range", rangeVal)
	d.renderEditableNatField(b, fieldDhcpLease, "DHCP Lease Time (sec)", d.getNatDhcpKV("Lease Limit (Seconds)", "Lease"))
	d.renderEditableNatField(b, fieldDhcpGW, "Default Gateway", d.getNatDhcpKV("Default Gateway Address", "Gateway", "GW"))
	d.renderEditableNatField(b, fieldDhcpDNS1, "DNS Server 1", d.getNatDhcpKV("DNS Server Address 1", "DNS"))
	d.renderEditableNatField(b, fieldDhcpDNS2, "DNS Server 2", d.getNatDhcpKV("DNS Server Address 2", "DNS2"))
	d.renderEditableNatField(b, fieldDhcpDomain, "Domain Name", d.getNatDhcpKV("Domain Name", "Domain"))

	b.WriteString("\n" + headerStyle.Render(tr("Status (動作状態)")) + "\n")
	statusKeys := []string{
		"Virtual Hub Name",
		"Allocated DHCP Clients",
		"Kernel-mode NAT is Active",
		"Raw IP mode NAT is Active",
		"NAT TCP/IP Sessions",
		"NAT UDP/IP Sessions",
		"NAT ICMP Sessions",
		"NAT DNS Sessions",
	}
	for _, k := range statusKeys {
		if v, ok := d.secureNatStatus[k]; ok {
			fmt.Fprintf(b, "  %-32s %s\n", k+":", statusBarStyle.Render(v))
		}
	}
	// Render any extra status keys not in the fixed list
	fixedMap := make(map[string]bool)
	for _, k := range statusKeys {
		fixedMap[k] = true
	}
	var extraKeys []string
	for k := range d.secureNatStatus {
		if !fixedMap[k] && k != "---" && k != "Item" {
			extraKeys = append(extraKeys, k)
		}
	}
	sort.Strings(extraKeys)
	for _, k := range extraKeys {
		fmt.Fprintf(b, "  %-32s %s\n", k+":", statusBarStyle.Render(d.secureNatStatus[k]))
	}
}

func (d hubDetailState) getNatHostKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.secureNatHost[k]; ok {
			return v
		}
	}
	return ""
}

func (d hubDetailState) getNatDhcpKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.secureNatDhcp[k]; ok {
			return v
		}
	}
	return ""
}

func (d hubDetailState) renderEditableNatField(b *strings.Builder, field editableSecureNATField, label, val string) {
	if ed, ok := d.secureNatEditedValues[field]; ok {
		val = ed + " " + selectedStyle.Render(tr("(変更あり)"))
	} else if val == "" {
		val = "(None)"
	}

	marker := "  "
	style := statusBarStyle
	if d.secureNatCursor == field {
		marker = "> "
		style = selectedStyle
	}

	if d.secureNatEditing && d.secureNatEditingField == field {
		fmt.Fprintf(b, "%s%-32s %s\n", marker, label+":", d.filterInput.View())
	} else {
		fmt.Fprintf(b, "%s%-32s %s\n", marker, label+":", style.Render(val))
	}
}

func (d hubDetailState) viewAccessList(b *strings.Builder) {
	switch {
	case d.accessLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.accessErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.accessErr.Error()) + "\n")
	case len(d.access.Rows) == 0:
		b.WriteString(dimStyle.Render(tr("(アクセスリストルールがありません)")) + "\n")
	default:
		b.WriteString(renderMultiColumnTable(d.access, d.accessCursor))
		if d.accessCursor >= 0 && d.accessCursor < len(d.access.Rows) {
			row := d.access.Rows[d.accessCursor]
			b.WriteString("\n" + headerStyle.Render(tr("選択中のルール詳細")) + "\n")
			keys := []string{"ID", "Action", "Status", "Priority", "Memo", "Contents", "Unique ID"}
			for _, k := range keys {
				if v, ok := row[k]; ok && v != "" {
					fmt.Fprintf(b, "  %-12s %s\n", k+":", v)
				}
			}
		}
	}
	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("選択"),
		"c", tr("追加"),
		"d", tr("削除"),
		"o", tr("有効化"),
		"f", tr("無効化"),
	))
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

// --- messages ---

type secureNatLoadedMsg struct {
	hubName   string
	status    vpncmd.KeyValue
	hubStatus vpncmd.KeyValue
	host      vpncmd.KeyValue
	dhcp      vpncmd.KeyValue
	err       error
}

type secureNatActionResultMsg struct {
	action string
	err    error
}

type accessLoadedMsg struct {
	hubName string
	table   vpncmd.Table
	err     error
}

type accessActionResultMsg struct {
	action string
	id     string
	err    error
}

// --- commands ---

func (m Model) fetchSecureNAT(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		time.Sleep(300 * time.Millisecond)
		status, statusErr := client.SecureNatStatusGet(ctx, target)
		if statusErr != nil {
			status = nil
		}
		hubStatus, _ := client.StatusGet(ctx, target)
		host, _ := client.SecureNatHostGet(ctx, target)
		dhcp, _ := client.DhcpGet(ctx, target)
		return secureNatLoadedMsg{hubName: hub, status: status, hubStatus: hubStatus, host: host, dhcp: dhcp, err: statusErr}
	}
}

func (m Model) setSecureNatEnabled(p config.Profile, hub string, enabled bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	action := tr("SecureNAT disabled")
	if enabled {
		action = tr("SecureNAT enabled")
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if enabled {
			err = client.SecureNatEnable(ctx, target)
		} else {
			err = client.SecureNatDisable(ctx, target)
		}
		return secureNatActionResultMsg{action: action, err: err}
	}
}

func (m Model) setDhcpEnabled(p config.Profile, hub string, enabled bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	action := tr("Virtual DHCP 無効化")
	if enabled {
		action = tr("Virtual DHCP 有効化")
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if enabled {
			err = client.DhcpEnable(ctx, target)
		} else {
			err = client.DhcpDisable(ctx, target)
		}
		return secureNatActionResultMsg{action: action, err: err}
	}
}

func (m Model) setSecureNatHost(p config.Profile, hub, ip, mask string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.SecureNatHostSet(ctx, target, vpncmd.SecureNatHostOptions{IP: ip, Mask: mask})
		return secureNatActionResultMsg{action: tr("仮想ホスト IP 設定"), err: err}
	}
}

func (m Model) setDhcpRange(p config.Profile, hub, startIp, endIp string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.DhcpSet(ctx, target, vpncmd.DhcpSetOptions{Start: startIp, End: endIp})
		return secureNatActionResultMsg{action: tr("DHCP 範囲設定"), err: err}
	}
}

func (m Model) fetchAccessList(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.AccessList(ctx, target)
		return accessLoadedMsg{hubName: hub, table: table, err: err}
	}
}

func (m Model) deleteAccessRule(p config.Profile, hub, id string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.AccessDelete(ctx, target, id)
		return accessActionResultMsg{action: tr("削除"), id: id, err: err}
	}
}

func (m Model) setAccessRuleEnabled(p config.Profile, hub, id string, enabled bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	action := tr("無効化")
	if enabled {
		action = tr("有効化")
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if enabled {
			err = client.AccessEnable(ctx, target, id)
		} else {
			err = client.AccessDisable(ctx, target, id)
		}
		return accessActionResultMsg{action: action, id: id, err: err}
	}
}

// --- key handling ---

func (m Model) handleHubACLKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hubDetail.accessCursor > 0 {
			m.hubDetail.accessCursor--
		}
	case "down", "j":
		if m.hubDetail.accessCursor < len(m.hubDetail.access.Rows)-1 {
			m.hubDetail.accessCursor++
		}
	case "c", "C", "a", "A":
		m.aclForm.Reset()
		m.screen = screenACLForm
		m.status = ""
		return m, nil
	case "e", "E", "enter":
		if id, ok := m.hubDetail.currentAccessID(); ok && m.hubDetail.accessCursor < len(m.hubDetail.access.Rows) {
			row := m.hubDetail.access.Rows[m.hubDetail.accessCursor]
			m.aclForm.LoadRule(id, row)
			m.screen = screenACLForm
			m.status = ""
			return m, nil
		}
	case "d":
		if id, ok := m.hubDetail.currentAccessID(); ok {
			m.confirm.Show(confirmDeleteAccessRule, id, fmt.Sprintf(tr("アクセスリストルール %q を削除しますか?"), id))
		}
	case "o":
		if id, ok := m.hubDetail.currentAccessID(); ok {
			m.status = fmt.Sprintf(tr("ルール %q を有効化しています..."), id)
			m.statusErr = false
			return m, m.setAccessRuleEnabled(m.hubDetail.profile, m.hubDetail.hubName, id, true)
		}
	case "f":
		if id, ok := m.hubDetail.currentAccessID(); ok {
			m.status = fmt.Sprintf(tr("ルール %q を無効化しています..."), id)
			m.statusErr = false
			return m, m.setAccessRuleEnabled(m.hubDetail.profile, m.hubDetail.hubName, id, false)
		}
	}
	return m, nil
}

// --- cascade: messages, commands, key handling ---

type cascadeLoadedMsg struct {
	hubName string
	table   vpncmd.Table
	err     error
}

func (m Model) addAccessRule(p config.Profile, hub string, opts vpncmd.AccessAddOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.AccessAdd(ctx, target, opts)
		return accessActionResultMsg{action: tr("追加"), id: opts.Memo, err: err}
	}
}

func (m Model) fetchCascade(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.CascadeList(ctx, target)
		return cascadeLoadedMsg{hubName: hub, table: table, err: err}
	}
}

func (m Model) deleteCascade(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.CascadeDelete(ctx, target, name)
		return cascadeActionResultMsg{action: tr("削除"), name: name, err: err}
	}
}

func (m Model) setCascadeOnline(p config.Profile, hub, name string, online bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	action := tr("オフライン化")
	if online {
		action = tr("オンライン化")
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.CascadeSetOnline(ctx, target, name, online)
		return cascadeActionResultMsg{action: action, name: name, err: err}
	}
}

func (m Model) handleHubCascadeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hubDetail.cascadeCursor > 0 {
			m.hubDetail.cascadeCursor--
		}
	case "down", "j":
		if m.hubDetail.cascadeCursor < len(m.hubDetail.cascade.Rows)-1 {
			m.hubDetail.cascadeCursor++
		}
	case "c", "C", "a", "A":
		m.cascadeForm.Reset()
		m.screen = screenCascadeForm
		m.status = ""
		return m, nil
	case "e", "E", "enter":
		if name, ok := m.hubDetail.currentCascadeName(); ok && m.hubDetail.cascadeCursor < len(m.hubDetail.cascade.Rows) {
			row := m.hubDetail.cascade.Rows[m.hubDetail.cascadeCursor]
			hostPort := row["Destination VPN Server"]
			host := hostPort
			port := 443
			if parts := strings.Split(hostPort, ":"); len(parts) == 2 {
				host = parts[0]
				if p, err := strconv.Atoi(parts[1]); err == nil {
					port = p
				}
			}
			targetHub := row["Virtual Hub"]
			user := row["User Name"]
			m.cascadeForm.LoadCascade(name, host, port, targetHub, user)
			m.screen = screenCascadeForm
			m.status = ""
			return m, nil
		}
	case "d":
		if name, ok := m.hubDetail.currentCascadeName(); ok {
			m.confirm.Show(confirmDeleteCascade, name, fmt.Sprintf(tr("カスケード接続 %q を削除しますか?"), name))
		}
	case "o":
		if name, ok := m.hubDetail.currentCascadeName(); ok {
			m.status = fmt.Sprintf(tr("カスケード接続 %q をオンライン化しています..."), name)
			m.statusErr = false
			return m, m.setCascadeOnline(m.hubDetail.profile, m.hubDetail.hubName, name, true)
		}
	case "f":
		if name, ok := m.hubDetail.currentCascadeName(); ok {
			m.status = fmt.Sprintf(tr("カスケード接続 %q をオフライン化しています..."), name)
			m.statusErr = false
			return m, m.setCascadeOnline(m.hubDetail.profile, m.hubDetail.hubName, name, false)
		}
	}
	return m, nil
}

func (d hubDetailState) viewCascade(b *strings.Builder) {
	switch {
	case d.cascadeLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.cascadeErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.cascadeErr.Error()) + "\n")
	case len(d.cascade.Rows) == 0:
		b.WriteString(dimStyle.Render(tr("(カスケード接続がありません)")) + "\n")
	default:
		b.WriteString(renderMultiColumnTable(d.cascade, d.cascadeCursor))
		if d.cascadeCursor >= 0 && d.cascadeCursor < len(d.cascade.Rows) {
			row := d.cascade.Rows[d.cascadeCursor]
			b.WriteString("\n" + headerStyle.Render(tr("選択中のカスケード詳細")) + "\n")
			keys := []string{"Setting Name", "Status", "Destination VPN Server", "Virtual Hub", "Established at"}
			for _, k := range keys {
				if v, ok := row[k]; ok && v != "" {
					fmt.Fprintf(b, "  %-24s %s\n", k+":", v)
				}
			}
		}
	}
	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("選択"),
		"c", tr("作成"),
		"d", tr("削除"),
		"o", tr("オンライン化"),
		"f", tr("オフライン化"),
	))
	b.WriteString("\n" + renderHelp(
		"←/→/Tab", tr("タブ切替"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
}

func renderHubTabs(active hubTab) string {
	tabLabels := hubTabLabels()
	var rendered []string
	sep := tabSepStyle.Render("|")
	for i, label := range tabLabels {
		if hubTab(i) == active {
			rendered = append(rendered, activeTabStyle.Render(label))
		} else {
			rendered = append(rendered, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(rendered, sep)
}
