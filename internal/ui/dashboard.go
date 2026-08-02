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
	bridges   vpncmd.Table
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

	b.WriteString("\n" + headerStyle.Render(tr("ローカルブリッジ一覧 (物理NIC/tap)")) + "\n")
	if len(d.bridges.Rows) == 0 {
		b.WriteString(dimStyle.Render(tr("  (ローカルブリッジが設定されていません)")) + "\n")
	} else {
		b.WriteString(renderMultiColumnTable(d.bridges, -1))
	}

	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("Hub選択"),
		"Enter", tr("Hub詳細"),
		"c", tr("Hub作成"),
		"d", tr("Hub削除"),
		"p", tr("Hubパスワード設定"),
		"l", tr("リスナー管理"),
		"b", tr("ローカルブリッジ"),
		"Esc", tr("戻る"),
		"r", tr("更新"),
		"q", tr("終了"),
	))
	return b.String()
}

func renderTable(table vpncmd.Table, cursor int) string {
	if len(table.Rows) == 0 {
		return dimStyle.Render(tr("(項目がありません)")) + "\n"
	}
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

func renderMultiColumnTable(table vpncmd.Table, cursor int) string {
	if len(table.Rows) == 0 {
		return dimStyle.Render(tr("(項目がありません)")) + "\n"
	}

	headersToDisplay := []string{"ID", "Action", "Status", "Priority", "Memo", "Contents", "Setting Name", "Destination VPN Server", "Virtual Hub", "Established at"}
	activeHeaders := make([]string, 0)
	widths := make(map[string]int)

	for _, h := range headersToDisplay {
		for _, th := range table.Headers {
			if strings.EqualFold(th, h) {
				activeHeaders = append(activeHeaders, th)
				widths[th] = len(th)
				break
			}
		}
	}
	if len(activeHeaders) == 0 {
		return renderTable(table, cursor)
	}

	for _, row := range table.Rows {
		for _, h := range activeHeaders {
			if v, ok := row[h]; ok {
				if len(v) > widths[h] {
					widths[h] = len(v)
				}
			}
		}
	}

	var b strings.Builder
	// Render Header
	b.WriteString("  " + headerStyle.Render(formatHeaderRow(activeHeaders, widths)) + "\n")

	// Render Rows
	for ri, row := range table.Rows {
		marker := "  "
		style := statusBarStyle
		if ri == cursor {
			marker = "> "
			style = selectedStyle
		}
		var rowBuf strings.Builder
		for i, h := range activeHeaders {
			v := row[h]
			w := widths[h]
			if i < len(activeHeaders)-1 {
				w += 2
			}
			fmt.Fprintf(&rowBuf, "%-*s", w, v)
		}
		b.WriteString(marker + style.Render(rowBuf.String()) + "\n")
	}
	return b.String()
}

func formatHeaderRow(headers []string, widths map[string]int) string {
	var b strings.Builder
	for i, h := range headers {
		w := widths[h]
		if i < len(headers)-1 {
			w += 2
		}
		fmt.Fprintf(&b, "%-*s", w, h)
	}
	return b.String()
}

func writeKV(b *strings.Builder, kv vpncmd.KeyValue) {
	if len(kv) == 0 {
		b.WriteString(dimStyle.Render(tr("(情報なし)")) + "\n")
		return
	}
	keys := make([]string, 0, len(kv))
	maxLen := 0
	for k := range kv {
		if k != "---" && k != "Item" {
			keys = append(keys, k)
			if len(k) > maxLen {
				maxLen = len(k)
			}
		}
	}
	sort.Strings(keys)
	maxLen += 2 // account for ":" colon and padding
	for _, k := range keys {
		fmt.Fprintf(b, "  %-*s %s\n", maxLen, k+":", kv[k])
	}
}
