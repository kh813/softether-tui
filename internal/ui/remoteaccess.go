package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

type editableRemoteAccessField int

const (
	fieldL2TP editableRemoteAccessField = iota
	fieldL2TPRaw
	fieldEtherIP
	fieldIPsecPSK
	fieldIPsecDefaultHub
	fieldOpenVPN
	fieldOpenVPNPorts
	fieldDDNSHostname
	editableRemoteAccessFieldCount
)

type remoteAccessState struct {
	profile config.Profile
	ipsec   vpncmd.KeyValue
	openvpn vpncmd.KeyValue
	ddns    vpncmd.KeyValue
	loading bool
	err     error

	cursor       editableRemoteAccessField
	editing      bool
	editingField editableRemoteAccessField
	input        textinput.Model
	editedValues map[editableRemoteAccessField]string
	dirty        bool
}

type remoteAccessLoadedMsg struct {
	ipsec   vpncmd.KeyValue
	openvpn vpncmd.KeyValue
	ddns    vpncmd.KeyValue
	err     error
}

func (m Model) fetchRemoteAccess(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ipsec, err := client.IPsecGet(ctx, target)
		if err != nil {
			return remoteAccessLoadedMsg{err: err}
		}
		openvpn, _ := client.OpenVpnGet(ctx, target)
		ddns, _ := client.DynamicDnsGetStatus(ctx, target)
		return remoteAccessLoadedMsg{ipsec: ipsec, openvpn: openvpn, ddns: ddns, err: nil}
	}
}

func (d remoteAccessState) getValue(f editableRemoteAccessField) string {
	if v, ok := d.editedValues[f]; ok {
		return v
	}
	switch f {
	case fieldL2TP:
		for k, v := range d.ipsec {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "raw") {
				continue
			}
			if strings.Contains(lk, "l2tp") {
				return v
			}
		}
		return "Disabled"
	case fieldL2TPRaw:
		for k, v := range d.ipsec {
			if strings.Contains(strings.ToLower(k), "raw") {
				return v
			}
		}
		return "Disabled"
	case fieldEtherIP:
		for k, v := range d.ipsec {
			if strings.Contains(strings.ToLower(k), "etherip") {
				return v
			}
		}
		return "Disabled"
	case fieldIPsecPSK:
		for k, v := range d.ipsec {
			if strings.Contains(strings.ToLower(k), "pre-shared") || strings.Contains(strings.ToLower(k), "psk") {
				return v
			}
		}
		return ""
	case fieldIPsecDefaultHub:
		for k, v := range d.ipsec {
			if strings.Contains(strings.ToLower(k), "default virtual hub") || strings.Contains(strings.ToLower(k), "default hub") {
				return v
			}
		}
		return "DEFAULT"
	case fieldOpenVPN:
		for k, v := range d.openvpn {
			if strings.Contains(strings.ToLower(k), "openvpn server function") || strings.Contains(strings.ToLower(k), "openvpn") {
				return v
			}
		}
		return "Disabled"
	case fieldOpenVPNPorts:
		for k, v := range d.openvpn {
			if strings.Contains(strings.ToLower(k), "port") {
				return v
			}
		}
		return "1194"
	case fieldDDNSHostname:
		for k, v := range d.ddns {
			if strings.Contains(strings.ToLower(k), "(hostname)") {
				return v
			}
		}
		for k, v := range d.ddns {
			if strings.Contains(strings.ToLower(k), "hostname") || strings.Contains(strings.ToLower(k), "host name") {
				return v
			}
		}
		return ""
	}
	return ""
}

func (d remoteAccessState) isEnabled(f editableRemoteAccessField) bool {
	v := strings.ToLower(d.getValue(f))
	return v == "enabled" || v == "yes" || v == "true"
}

func (d remoteAccessState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(tr("リモートアクセス設定 (IPsec / OpenVPN / DDNS)")))
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
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更/切替"), "s", tr("保存 (Save)"), "n", tr("変更を破棄 (Cancel)")))
	} else {
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更/切替"), "r", tr("更新"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d remoteAccessState) renderSections(b *strings.Builder) {
	b.WriteString(headerStyle.Render(tr("IPsec / L2TP / EtherIP 設定")) + "\n")
	d.renderToggleRow(b, fieldL2TP, tr("L2TP サーバー機能 (IPsec)"))
	d.renderToggleRow(b, fieldL2TPRaw, tr("L2TP RAW (暗号化なし)"))
	d.renderToggleRow(b, fieldEtherIP, tr("EtherIP / L2TPv3 機能"))
	d.renderInputRow(b, fieldIPsecPSK, tr("IPsec 事前共有鍵 (PSK)"))
	d.renderInputRow(b, fieldIPsecDefaultHub, tr("デフォルト Virtual HUB"))

	b.WriteString("\n" + headerStyle.Render(tr("OpenVPN 互換サーバー設定")) + "\n")
	d.renderToggleRow(b, fieldOpenVPN, tr("OpenVPN サーバー機能"))
	d.renderInputRow(b, fieldOpenVPNPorts, tr("OpenVPN UDP ポート一覧"))

	b.WriteString("\n" + headerStyle.Render(tr("ダイナミック DNS (DDNS) 状態")) + "\n")
	d.renderInputRow(b, fieldDDNSHostname, tr("DDNS ホスト名 (変更可能)"))

	// Read-only Global IPs from DDNS
	ipv4 := ""
	ipv6 := ""
	for k, v := range d.ddns {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "ipv4") {
			ipv4 = v
		} else if strings.Contains(lk, "ipv6") {
			ipv6 = v
		}
	}
	if ipv4 != "" {
		fmt.Fprintf(b, "    %-28s: %s\n", tr("グローバル IPv4"), dimStyle.Render(ipv4))
	}
	if ipv6 != "" {
		fmt.Fprintf(b, "    %-28s: %s\n", tr("グローバル IPv6"), dimStyle.Render(ipv6))
	}
}

func (d remoteAccessState) renderToggleRow(b *strings.Builder, f editableRemoteAccessField, label string) {
	cursor := "  "
	if d.cursor == f {
		cursor = "> "
	}
	en := d.isEnabled(f)
	var badge string
	if en {
		badge = selectedStyle.Render("[ " + tr("有効 (Enabled)") + " ]")
	} else {
		badge = statusBarStyle.Render("[ " + tr("無効 (Disabled)") + " ]")
	}
	fmt.Fprintf(b, "%s%-30s: %s\n", cursor, label, badge)
}

func (d remoteAccessState) renderInputRow(b *strings.Builder, f editableRemoteAccessField, label string) {
	cursor := "  "
	if d.cursor == f {
		cursor = "> "
	}
	if d.editing && d.editingField == f {
		fmt.Fprintf(b, "%s%-30s: %s\n", cursor, label, d.input.View())
		return
	}
	val := d.getValue(f)
	if val == "" {
		val = tr("(未設定)")
	}
	style := statusBarStyle
	if d.cursor == f {
		style = selectedStyle
	}
	fmt.Fprintf(b, "%s%-30s: %s\n", cursor, label, style.Render(val))
}

func (d remoteAccessState) isToggleField(f editableRemoteAccessField) bool {
	switch f {
	case fieldL2TP, fieldL2TPRaw, fieldEtherIP, fieldOpenVPN:
		return true
	default:
		return false
	}
}

type remoteAccessSavedMsg struct {
	err error
}

func (m Model) handleRemoteAccessKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.remoteAccess

	if d.editing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.input.Value())
			if d.editedValues == nil {
				d.editedValues = make(map[editableRemoteAccessField]string)
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
		m.screen = screenDashboard
		return m, nil

	case "n", "N":
		if d.dirty {
			m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
			return m, nil
		}

	case "r", "R":
		d.loading = true
		return m, m.fetchRemoteAccess(d.profile)

	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}

	case "down", "j":
		if d.cursor < editableRemoteAccessFieldCount-1 {
			d.cursor++
		}

	case "enter", " ":
		if d.isToggleField(d.cursor) {
			if d.editedValues == nil {
				d.editedValues = make(map[editableRemoteAccessField]string)
			}
			if d.isEnabled(d.cursor) {
				d.editedValues[d.cursor] = "Disabled"
			} else {
				d.editedValues[d.cursor] = "Enabled"
			}
			d.dirty = true
			return m, nil
		}

		d.editing = true
		d.editingField = d.cursor
		ti := textinput.New()
		ti.SetValue(d.getValue(d.cursor))
		ti.Focus()
		d.input = ti
		return m, nil

	case "s", "S":
		if d.dirty {
			return m.saveRemoteAccessChanges()
		}
	}
	return m, nil
}

func (m Model) saveRemoteAccessChanges() (tea.Model, tea.Cmd) {
	d := &m.remoteAccess
	p := d.profile
	client := m.client
	target := m.targetFromProfile(p)

	ipsecOpts := vpncmd.IPsecOptions{
		L2TP:       d.isEnabled(fieldL2TP),
		L2TPRaw:    d.isEnabled(fieldL2TPRaw),
		EtherIP:    d.isEnabled(fieldEtherIP),
		PSK:        d.getValue(fieldIPsecPSK),
		DefaultHub: d.getValue(fieldIPsecDefaultHub),
	}

	openVpnEnabled := d.isEnabled(fieldOpenVPN)
	portsStr := d.getValue(fieldOpenVPNPorts)
	var ports []int
	for _, part := range strings.Split(portsStr, ",") {
		pStr := strings.TrimSpace(part)
		if port, err := strconv.Atoi(pStr); err == nil && port > 0 {
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		ports = []int{1194}
	}

	ddnsHostname := ""
	if v, ok := d.editedValues[fieldDDNSHostname]; ok {
		ddnsHostname = strings.TrimSpace(v)
	}

	m.status = tr("リモートアクセス設定を保存しています...")
	m.statusErr = false

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := client.IPsecEnable(ctx, target, ipsecOpts); err != nil {
			return remoteAccessSavedMsg{err: err}
		}

		if err := client.OpenVpnEnable(ctx, target, openVpnEnabled, ports); err != nil {
			return remoteAccessSavedMsg{err: err}
		}

		if ddnsHostname != "" {
			if err := client.DynamicDnsSetHostname(ctx, target, ddnsHostname); err != nil {
				return remoteAccessSavedMsg{err: err}
			}
		}

		return remoteAccessSavedMsg{err: nil}
	}
}
