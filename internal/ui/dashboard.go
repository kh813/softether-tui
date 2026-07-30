package ui

import (
	"fmt"
	"sort"
	"strings"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

// dashboardState is the server dashboard screen (app_specs.md 8.2 ②).
type dashboardState struct {
	profile   config.Profile
	info      vpncmd.KeyValue
	status    vpncmd.KeyValue
	hubs      vpncmd.Table
	hubCursor int
	loading   bool
	err       error
}

func (d dashboardState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(fmt.Sprintf("%s (%s)", d.profile.Name, d.profile.Address())))
	b.WriteString(strings.Repeat("─", 60) + "\n")

	if d.loading {
		b.WriteString(tr("接続確認中...") + "\n")
		return b.String()
	}
	if d.err != nil {
		b.WriteString(errorStyle.Render(tr("接続エラー: ")+d.err.Error()) + "\n")
		return b.String()
	}

	// Compact server summary header
	product := d.info["Product Name"]
	if product == "" {
		product = "SoftEther VPN Server"
	}
	ver := d.info["Version"]
	osType := d.info["Type of Operating System"]
	hubsCount := d.status["Number of Virtual Hubs"]
	sessionsCount := d.status["Number of Sessions"]

	fmt.Fprintf(&b, "%s %s (%s)\n", product, ver, osType)
	if hubsCount != "" || sessionsCount != "" {
		fmt.Fprintf(&b, tr("Hub数: %s   総セッション数: %s\n"), hubsCount, sessionsCount)
	}
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	b.WriteString(headerStyle.Render(tr("Hub一覧")) + "\n")
	b.WriteString(renderTable(d.hubs, d.hubCursor))

	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:Hub選択  Enter:Hub詳細  a:Hub作成  d:Hub削除  l:リスナー管理  b:ローカルブリッジ  Esc:戻る  r:更新  q:終了")))
	return b.String()
}

// renderTable renders a generic vpncmd.Table (used for HubList and, from M3
// onward, UserList/SessionList) as a simple column-aligned list, since the
// exact column names vpncmd returns are not hard-coded (see
// vpncmd_commands.md caveats).
func renderTable(table vpncmd.Table, cursor int) string {
	if len(table.Rows) == 0 {
		return dimStyle.Render(tr("(項目がありません)")) + "\n"
	}

	// For HubList and simple list views, extract only the first primary column (e.g. "Virtual Hub Name")
	mainHeader := table.Headers[0]

	var b strings.Builder
	for ri, row := range table.Rows {
		marker := "  "
		style := statusBarStyle
		if ri == cursor {
			marker = "> "
			style = selectedStyle
		}
		val := row[mainHeader]
		if val == "" {
			val = table.NameOf(row)
		}
		b.WriteString(marker + style.Render(val) + "\n")
	}
	return b.String()
}

func writeKV(b *strings.Builder, kv vpncmd.KeyValue) {
	if len(kv) == 0 {
		b.WriteString(dimStyle.Render(tr("(情報なし)")) + "\n")
		return
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "  %-28s %s\n", k+":", kv[k])
	}
}
