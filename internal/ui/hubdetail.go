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

type hubTab int

const (
	hubTabOverview hubTab = iota
	hubTabUsers
	hubTabGroups
	hubTabSessions
	hubTabLog
	hubTabSecureNAT
	hubTabACL
	hubTabCascade
	hubTabCount
)

func hubTabLabels() [hubTabCount]string {
	return [hubTabCount]string{
		tr("概要"), tr("ユーザー"), tr("グループ"), tr("セッション"), tr("セキュリティログ"), "SecureNAT", "ACL", tr("カスケード"),
	}
}

// hubDetailState is the Hub detail screen (app_specs.md 8.2 ③). Overview,
// Users and Groups tabs have real data wired up (M2/M3); the rest are
// placeholders for M4-M6 to fill in.
type hubDetailState struct {
	profile config.Profile
	hubName string
	tab     hubTab

	info    vpncmd.KeyValue
	loading bool
	err     error

	users        vpncmd.Table
	usersLoaded  bool
	usersLoading bool
	usersErr     error
	userCursor   int
	userFilter   string
	filtering    bool
	filterInput  textinput.Model

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

	secureNatStatus  vpncmd.KeyValue
	secureNatHost    vpncmd.KeyValue
	secureNatLoaded  bool
	secureNatLoading bool
	secureNatErr     error

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
	case hubTabUsers:
		d.viewUsers(&b)
	case hubTabGroups:
		d.viewGroups(&b)
	case hubTabSessions:
		d.viewSessions(&b)
	case hubTabLog:
		d.viewLog(&b)
	case hubTabSecureNAT:
		d.viewSecureNAT(&b)
	case hubTabACL:
		d.viewAccessList(&b)
	case hubTabCascade:
		d.viewCascade(&b)
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
	b.WriteString("\n" + dimStyle.Render(tr("Tab/Shift+Tab:タブ切替  o:オンライン化  f:オフライン化  r:更新  Esc:戻る  q:終了")))
}

func (d hubDetailState) viewUsers(b *strings.Builder) {
	if d.filtering {
		fmt.Fprintf(b, tr("検索: %s\n\n"), d.filterInput.View())
	} else if d.userFilter != "" {
		fmt.Fprintf(b, tr("検索: %s\n\n"), d.userFilter)
	}

	switch {
	case d.usersLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.usersErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.usersErr.Error()) + "\n")
	default:
		rows := d.filteredUsers()
		b.WriteString(renderTable(vpncmd.Table{Headers: d.users.Headers, Rows: rows}, d.userCursor))
		fmt.Fprintf(b, tr("%d件 / 全%d件\n"), len(rows), len(d.users.Rows))
	}

	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  /:検索  a:追加  d:削除  p:パスワード再設定  g:グループ変更  e:有効期限設定")))
	b.WriteString("\n" + dimStyle.Render(tr("Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
}

func (d hubDetailState) viewGroups(b *strings.Builder) {
	switch {
	case d.groupsLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.groupsErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.groupsErr.Error()) + "\n")
	default:
		b.WriteString(renderTable(d.groups, d.groupCursor))
	}
	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  a:追加  d:削除  Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
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

	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  x:切断  +/-:更新間隔変更  r:手動更新")))
	b.WriteString("\n" + dimStyle.Render(tr("Tab/Shift+Tab:タブ切替  Esc:戻る  q:終了")))
}

func (d hubDetailState) viewLog(b *strings.Builder) {
	switch {
	case d.logLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.logErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.logErr.Error()) + "\n")
	default:
		writeKV(b, d.logInfo)
	}
	b.WriteString("\n" + dimStyle.Render(tr("(ログファイル内容の閲覧・保存設定の変更は今後のマイルストーンで対応予定)")))
	b.WriteString("\n" + dimStyle.Render(tr("Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
}

func (d hubDetailState) viewSecureNAT(b *strings.Builder) {
	switch {
	case d.secureNatLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.secureNatErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.secureNatErr.Error()) + "\n")
	default:
		b.WriteString(headerStyle.Render(tr("状態")) + "\n")
		writeKV(b, d.secureNatStatus)
		b.WriteString("\n" + headerStyle.Render(tr("仮想ホスト設定")) + "\n")
		writeKV(b, d.secureNatHost)
	}
	b.WriteString("\n" + dimStyle.Render(tr("o:有効化  f:無効化  i:仮想ホストIP設定  s:DHCP範囲設定  Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
}

func (d hubDetailState) viewAccessList(b *strings.Builder) {
	switch {
	case d.accessLoading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.accessErr != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.accessErr.Error()) + "\n")
	default:
		b.WriteString(renderTable(d.access, d.accessCursor))
	}
	b.WriteString("\n" + dimStyle.Render(tr("(ルール追加は今後のマイルストーンで対応予定。既存ルールの削除/有効/無効のみ対応)")))
	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  d:削除  o:有効化  f:無効化  Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
}

// --- messages ---

type secureNatLoadedMsg struct {
	hubName string
	status  vpncmd.KeyValue
	host    vpncmd.KeyValue
	err     error
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
		status, err := client.SecureNatStatusGet(ctx, target)
		if err != nil {
			return secureNatLoadedMsg{hubName: hub, err: err}
		}
		host, err := client.SecureNatHostGet(ctx, target)
		if err != nil {
			return secureNatLoadedMsg{hubName: hub, err: err}
		}
		return secureNatLoadedMsg{hubName: hub, status: status, host: host}
	}
}

func (m Model) setSecureNatEnabled(p config.Profile, hub string, enabled bool) tea.Cmd {
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
			err = client.SecureNatEnable(ctx, target)
		} else {
			err = client.SecureNatDisable(ctx, target)
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

type cascadeActionResultMsg struct {
	action string
	name   string
	err    error
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
	default:
		b.WriteString(renderTable(d.cascade, d.cascadeCursor))
	}
	b.WriteString("\n" + dimStyle.Render(tr("(新規カスケード接続の作成は今後のマイルストーンで対応予定。既存接続の削除/オンライン/オフラインのみ対応)")))
	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  d:削除  o:オンライン化  f:オフライン化  Tab/Shift+Tab:タブ切替  r:更新  Esc:戻る  q:終了")))
}

func renderHubTabs(active hubTab) string {
	tabLabels := hubTabLabels()
	labels := make([]string, len(tabLabels))
	for i, label := range tabLabels {
		if hubTab(i) == active {
			labels[i] = selectedStyle.Render("[" + label + "]")
		} else {
			labels[i] = dimStyle.Render(label)
		}
	}
	return strings.Join(labels, " ")
}
