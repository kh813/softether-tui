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

type editableSecureNATField int

const (
	fieldSecureNAT editableSecureNATField = iota
	fieldNatIP
	fieldNatMask
	fieldNatMAC
	fieldNatMTU
	fieldDHCP
	fieldDhcpRange
	fieldDhcpLease
	fieldDhcpGW
	fieldDhcpDNS1
	fieldDhcpDNS2
	fieldDhcpDomain
	editableSecureNATFieldCount
)

type secureNatDetailState struct {
	profile config.Profile
	hubName string
	status  vpncmd.KeyValue
	host    vpncmd.KeyValue
	dhcp    vpncmd.KeyValue
	loading bool
	err     error

	cursor       editableSecureNATField
	editing      bool
	editingField editableSecureNATField
	input        textinput.Model
	editedValues map[editableSecureNATField]string
	dirty        bool
}

type secureNatDetailLoadedMsg struct {
	hubName string
	status  vpncmd.KeyValue
	host    vpncmd.KeyValue
	dhcp    vpncmd.KeyValue
	err     error
}

func (m Model) fetchSecureNATDetail(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		status, err := client.SecureNatStatusGet(ctx, target)
		if err != nil {
			return secureNatDetailLoadedMsg{hubName: hub, err: err}
		}
		host, _ := client.SecureNatHostGet(ctx, target)
		dhcp, _ := client.DhcpGet(ctx, target)
		return secureNatDetailLoadedMsg{hubName: hub, status: status, host: host, dhcp: dhcp, err: nil}
	}
}

func (d secureNatDetailState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render(fmt.Sprintf(tr("SecureNAT 詳細設定 (Hub: %s)"), d.hubName)))
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
		b.WriteString("\n" + renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "Esc", tr("戻る"), "q", tr("終了")))
	}
	return b.String()
}

func (d secureNatDetailState) renderSections(b *strings.Builder) {
	b.WriteString(headerStyle.Render(tr("Virtual host settings (仮想ホスト・DHCP設定)")) + "\n")

	// Render editable fields
	d.renderEditableField(b, fieldNatIP, "IP Address", d.getHostKV("IP Address", "IP"))
	d.renderEditableField(b, fieldNatMask, "Subnet Mask", d.getHostKV("Subnet Mask", "Mask"))
	d.renderEditableField(b, fieldNatMAC, "MAC Address", d.getHostKV("MAC Address", "MAC"))

	startIp := d.getDhcpKV("Start Distribution Address Band", "Start")
	endIp := d.getDhcpKV("End Distribution Address Band", "End")
	rangeVal := startIp
	if endIp != "" {
		rangeVal = startIp + " - " + endIp
	}
	d.renderEditableField(b, fieldDhcpRange, "DHCP Range", rangeVal)
	d.renderEditableField(b, fieldDhcpLease, "DHCP Lease Time (sec)", d.getDhcpKV("Lease Limit (Seconds)", "Lease"))
	d.renderEditableField(b, fieldDhcpDNS1, "DNS Server 1", d.getDhcpKV("DNS Server Address 1", "DNS"))
	d.renderEditableField(b, fieldDhcpDNS2, "DNS Server 2", d.getDhcpKV("DNS Server Address 2", "DNS2"))
	d.renderEditableField(b, fieldDhcpDomain, "Domain Name", d.getDhcpKV("Domain Name", "Domain"))

	b.WriteString("\n" + headerStyle.Render(tr("Status (動作状態)")) + "\n")
	for k, v := range d.status {
		if k != "---" && k != "Item" {
			fmt.Fprintf(b, "  %-32s %s\n", k+":", statusBarStyle.Render(v))
		}
	}
}

func (d secureNatDetailState) getHostKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.host[k]; ok {
			return v
		}
	}
	return ""
}

func (d secureNatDetailState) getDhcpKV(keys ...string) string {
	for _, k := range keys {
		if v, ok := d.dhcp[k]; ok {
			return v
		}
	}
	return ""
}

func (d secureNatDetailState) renderEditableField(b *strings.Builder, field editableSecureNATField, label, val string) {
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
		fmt.Fprintf(b, "%s%-32s %s\n", marker, label+":", d.input.View())
	} else {
		fmt.Fprintf(b, "%s%-32s %s\n", marker, label+":", style.Render(val))
	}
}

func (m Model) handleSecureNATDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.secureNatDetail

	if d.editing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.input.Value())
			if d.editedValues == nil {
				d.editedValues = make(map[editableSecureNATField]string)
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
			d.editedValues = make(map[editableSecureNATField]string)
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
		if d.cursor < editableSecureNATFieldCount-1 {
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
			return m.saveSecureNATDetailChanges()
		}
	}
	return m, nil
}

func (d secureNatDetailState) getFieldValue(field editableSecureNATField) string {
	switch field {
	case fieldNatIP:
		return d.getHostKV("IP Address", "IP")
	case fieldNatMask:
		return d.getHostKV("Subnet Mask", "Mask")
	case fieldNatMAC:
		return d.getHostKV("MAC Address", "MAC")
	case fieldDhcpRange:
		startIp := d.getDhcpKV("Start Distribution Address Band", "Start")
		endIp := d.getDhcpKV("End Distribution Address Band", "End")
		if startIp != "" && endIp != "" {
			return startIp + " - " + endIp
		}
		return startIp
	case fieldDhcpLease:
		return d.getDhcpKV("Lease Limit (Seconds)", "Lease")
	case fieldDhcpDNS1:
		return d.getDhcpKV("DNS Server Address 1", "DNS")
	case fieldDhcpDNS2:
		return d.getDhcpKV("DNS Server Address 2", "DNS2")
	case fieldDhcpDomain:
		return d.getDhcpKV("Domain Name", "Domain")
	}
	return ""
}

func (m Model) saveSecureNATDetailChanges() (tea.Model, tea.Cmd) {
	d := &m.secureNatDetail
	p := d.profile
	hub := d.hubName
	var cmds []tea.Cmd

	ip := d.editedValues[fieldNatIP]
	mask := d.editedValues[fieldNatMask]
	mac := d.editedValues[fieldNatMAC]

	if ip != "" || mask != "" || mac != "" {
		hostOpts := vpncmd.SecureNatHostOptions{
			IP:   ip,
			Mask: mask,
			MAC:  mac,
		}
		cmds = append(cmds, m.setSecureNatHostOpts(p, hub, hostOpts))
	}

	rangeVal := d.editedValues[fieldDhcpRange]
	lease := d.editedValues[fieldDhcpLease]
	dns1 := d.editedValues[fieldDhcpDNS1]
	dns2 := d.editedValues[fieldDhcpDNS2]
	domain := d.editedValues[fieldDhcpDomain]

	if rangeVal != "" || lease != "" || dns1 != "" || dns2 != "" || domain != "" {
		startIp, endIp := "", ""
		if rangeVal != "" {
			parts := strings.Split(rangeVal, "-")
			startIp = strings.TrimSpace(parts[0])
			endIp = startIp
			if len(parts) > 1 {
				endIp = strings.TrimSpace(parts[1])
			}
		}
		dhcpOpts := vpncmd.DhcpSetOptions{
			Start:  startIp,
			End:    endIp,
			Expire: lease,
			DNS:    dns1,
			DNS2:   dns2,
			Domain: domain,
		}
		cmds = append(cmds, m.setDhcpOpts(p, hub, dhcpOpts))
	}

	d.dirty = false
	d.editedValues = make(map[editableSecureNATField]string)
	m.status = fmt.Sprintf(tr("Hub %q の SecureNAT 設定を保存しています..."), hub)
	m.statusErr = false

	cmds = append(cmds, m.fetchSecureNATDetail(p, hub))
	return m, tea.Batch(cmds...)
}

func (m Model) setSecureNatHostOpts(p config.Profile, hub string, opts vpncmd.SecureNatHostOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.SecureNatHostSet(ctx, target, opts)
		return secureNatActionResultMsg{action: tr("仮想ホスト設定変更"), err: err}
	}
}

func (m Model) setDhcpOpts(p config.Profile, hub string, opts vpncmd.DhcpSetOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.DhcpSet(ctx, target, opts)
		return secureNatActionResultMsg{action: tr("DHCP設定変更"), err: err}
	}
}
