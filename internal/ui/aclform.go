package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type aclFormField int

const (
	aclFieldPass aclFormField = iota
	aclFieldEnable
	aclFieldPriority
	aclFieldMemo
	aclFieldProtocol
	aclFieldSrcIP
	aclFieldDstIP
	aclFieldSrcPort
	aclFieldDstPort
	aclFieldTcpState
	aclFieldSrcUser
	aclFieldDstUser
	aclFieldSrcMAC
	aclFieldDstMAC
	aclFieldSave
	aclFieldCount
)

type aclForm struct {
	pass        bool // true = pass, false = discard
	enable      bool // true = enable, false = disable
	protocolIdx int  // 0: ALL(0), 1: ICMPv4(1), 2: TCP(6), 3: UDP(17), 4: ICMPv6(58)
	tcpStateIdx int  // 0: All(""), 1: Established("Established"), 2: Unestablished("Unestablished")

	inputs   [10]textinput.Model // priority, memo, srcIP, dstIP, srcPort, dstPort, srcUser, dstUser, srcMAC, dstMAC
	focus    aclFormField
	editing  bool
	targetID string // original rule ID if editing
	dirty    bool
}

var aclProtocolOrder = []struct {
	Name string
	Val  string
}{
	{Name: "ALL (0)", Val: "0"},
	{Name: "ICMPv4 (1)", Val: "1"},
	{Name: "TCP (6)", Val: "6"},
	{Name: "UDP (17)", Val: "17"},
	{Name: "ICMPv6 (58)", Val: "58"},
}

var aclTcpStateOrder = []struct {
	Name string
	Val  string
}{
	{Name: "All", Val: ""},
	{Name: "Established", Val: "Established"},
	{Name: "Unestablished", Val: "Unestablished"},
}

func newACLForm() *aclForm {
	f := &aclForm{
		pass:   true,
		enable: true,
	}
	placeholders := []string{
		"100",               // priority
		tr("ルール説明・メモ"),      // memo
		"0.0.0.0/0",         // srcIP
		"0.0.0.0/0",         // dstIP
		"0",                 // srcPort
		"0",                 // dstPort
		tr("送信元ユーザー名 (任意)"), // srcUser
		tr("送信先ユーザー名 (任意)"), // dstUser
		"00:00:00:00:00:00", // srcMAC
		"00:00:00:00:00:00", // dstMAC
	}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	f.inputs[0].SetValue("100")       // priority default
	f.inputs[2].SetValue("0.0.0.0/0") // srcIP default
	f.inputs[3].SetValue("0.0.0.0/0") // dstIP default
	f.inputs[4].SetValue("0")         // srcPort default
	f.inputs[5].SetValue("0")         // dstPort default
	f.setFocus(aclFieldPass)
	return f
}

func (f *aclForm) Reset() {
	f.editing = false
	f.targetID = ""
	f.pass = true
	f.enable = true
	f.protocolIdx = 0
	f.tcpStateIdx = 0
	f.dirty = false

	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.inputs[0].SetValue("100")
	f.inputs[2].SetValue("0.0.0.0/0")
	f.inputs[3].SetValue("0.0.0.0/0")
	f.inputs[4].SetValue("0")
	f.inputs[5].SetValue("0")
	f.setFocus(aclFieldPass)
}

func (f *aclForm) LoadRule(id string, row map[string]string) {
	f.editing = true
	f.targetID = id
	f.dirty = false

	action := strings.ToLower(row["Action"])
	f.pass = !strings.Contains(action, "discard")

	status := strings.ToLower(row["Status"])
	f.enable = !strings.Contains(status, "disable")

	f.inputs[0].SetValue(row["Priority"])
	f.inputs[1].SetValue(row["Memo"])

	// Parse Contents or specific fields if available
	contents := row["Contents"]
	f.parseContents(contents)

	f.setFocus(aclFieldPass)
}

func (f *aclForm) parseContents(contents string) {
	if contents == "" {
		return
	}
	cLower := strings.ToLower(contents)

	// Protocol
	switch {
	case strings.Contains(cLower, "(icmpv6)") || strings.Contains(cLower, "protocol: 58"):
		f.protocolIdx = 4
	case strings.Contains(cLower, "(icmp)") || strings.Contains(cLower, "protocol: 1"):
		f.protocolIdx = 1
	case strings.Contains(cLower, "(tcp)") || strings.Contains(cLower, "protocol: 6"):
		f.protocolIdx = 2
	case strings.Contains(cLower, "(udp)") || strings.Contains(cLower, "protocol: 17"):
		f.protocolIdx = 3
	default:
		f.protocolIdx = 0
	}

	// Parse fields from space-separated or key:value contents if present
	// SoftEther formats Contents as e.g. "(TCP) Src: 192.168.1.0/24:80 Dst: 10.0.0.1:443" or similar
	parts := strings.Fields(contents)
	for i, p := range parts {
		pLower := strings.ToLower(p)
		if strings.HasPrefix(pLower, "src:") || strings.HasPrefix(pLower, "src_ip:") {
			val := strings.TrimPrefix(p, "src:")
			val = strings.TrimPrefix(val, "src_ip:")
			if host, port, ok := parseIPAndPort(val); ok {
				f.inputs[2].SetValue(host)
				f.inputs[4].SetValue(port)
			} else {
				f.inputs[2].SetValue(val)
			}
		} else if strings.HasPrefix(pLower, "dst:") || strings.HasPrefix(pLower, "dst_ip:") {
			val := strings.TrimPrefix(p, "dst:")
			val = strings.TrimPrefix(val, "dst_ip:")
			if host, port, ok := parseIPAndPort(val); ok {
				f.inputs[3].SetValue(host)
				f.inputs[5].SetValue(port)
			} else {
				f.inputs[3].SetValue(val)
			}
		} else if strings.HasPrefix(pLower, "srcport:") || strings.HasPrefix(pLower, "sport:") {
			f.inputs[4].SetValue(strings.TrimPrefix(strings.TrimPrefix(p, "srcport:"), "sport:"))
		} else if strings.HasPrefix(pLower, "dstport:") || strings.HasPrefix(pLower, "dport:") {
			f.inputs[5].SetValue(strings.TrimPrefix(strings.TrimPrefix(p, "dstport:"), "dport:"))
		} else if strings.HasPrefix(pLower, "srcuser:") {
			f.inputs[6].SetValue(strings.TrimPrefix(p, "srcuser:"))
		} else if strings.HasPrefix(pLower, "dstuser:") {
			f.inputs[7].SetValue(strings.TrimPrefix(p, "dstuser:"))
		} else if strings.HasPrefix(pLower, "srcmac:") {
			f.inputs[8].SetValue(strings.TrimPrefix(p, "srcmac:"))
		} else if strings.HasPrefix(pLower, "dstmac:") {
			f.inputs[9].SetValue(strings.TrimPrefix(p, "dstmac:"))
		} else if strings.Contains(pLower, "established") {
			if strings.Contains(pLower, "unestablished") {
				f.tcpStateIdx = 2
			} else {
				f.tcpStateIdx = 1
			}
		}
		_ = i
	}
}

func parseIPAndPort(val string) (ip string, port string, ok bool) {
	if idx := strings.LastIndex(val, ":"); idx > 0 {
		return val[:idx], val[idx+1:], true
	}
	return val, "0", false
}

func (f *aclForm) setFocus(field aclFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	idx := f.inputIndex(field)
	if idx >= 0 && idx < len(f.inputs) {
		f.inputs[idx].Focus()
	}
}

func (f *aclForm) inputIndex(field aclFormField) int {
	switch field {
	case aclFieldPriority:
		return 0
	case aclFieldMemo:
		return 1
	case aclFieldSrcIP:
		return 2
	case aclFieldDstIP:
		return 3
	case aclFieldSrcPort:
		return 4
	case aclFieldDstPort:
		return 5
	case aclFieldSrcUser:
		return 6
	case aclFieldDstUser:
		return 7
	case aclFieldSrcMAC:
		return 8
	case aclFieldDstMAC:
		return 9
	default:
		return -1
	}
}

func (f *aclForm) IsDirty() bool {
	return f.dirty
}

func (f *aclForm) Build() (opts vpncmd.AccessAddOptions, err error) {
	prioStr := strings.TrimSpace(f.inputs[0].Value())
	prio, convErr := strconv.Atoi(prioStr)
	if convErr != nil || prio < 1 {
		err = fmt.Errorf(tr("優先度は1以上の数値で指定してください: %q"), prioStr)
		return
	}

	memo := strings.TrimSpace(f.inputs[1].Value())
	if memo == "" {
		err = errors.New(tr("ルール説明 (Memo) は必須です"))
		return
	}

	opts = vpncmd.AccessAddOptions{
		Pass:     f.pass,
		Memo:     memo,
		Priority: prio,
		Protocol: aclProtocolOrder[f.protocolIdx].Val,
		SrcIP:    strings.TrimSpace(f.inputs[2].Value()),
		DstIP:    strings.TrimSpace(f.inputs[3].Value()),
		SrcPort:  strings.TrimSpace(f.inputs[4].Value()),
		DstPort:  strings.TrimSpace(f.inputs[5].Value()),
		TcpState: aclTcpStateOrder[f.tcpStateIdx].Val,
		SrcUser:  strings.TrimSpace(f.inputs[6].Value()),
		DstUser:  strings.TrimSpace(f.inputs[7].Value()),
		SrcMAC:   strings.TrimSpace(f.inputs[8].Value()),
		DstMAC:   strings.TrimSpace(f.inputs[9].Value()),
	}
	return
}

func (f *aclForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % aclFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + aclFieldCount) % aclFieldCount)
		return nil
	case "enter", "space":
		if f.focus == aclFieldPass {
			f.pass = !f.pass
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldEnable {
			f.enable = !f.enable
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldProtocol {
			f.protocolIdx = (f.protocolIdx + 1) % len(aclProtocolOrder)
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldTcpState {
			f.tcpStateIdx = (f.tcpStateIdx + 1) % len(aclTcpStateOrder)
			f.dirty = true
			return nil
		}
	case "left", "h":
		if f.focus == aclFieldPass {
			f.pass = !f.pass
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldEnable {
			f.enable = !f.enable
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldProtocol {
			f.protocolIdx = (f.protocolIdx - 1 + len(aclProtocolOrder)) % len(aclProtocolOrder)
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldTcpState {
			f.tcpStateIdx = (f.tcpStateIdx - 1 + len(aclTcpStateOrder)) % len(aclTcpStateOrder)
			f.dirty = true
			return nil
		}
	case "right", "l":
		if f.focus == aclFieldPass {
			f.pass = !f.pass
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldEnable {
			f.enable = !f.enable
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldProtocol {
			f.protocolIdx = (f.protocolIdx + 1) % len(aclProtocolOrder)
			f.dirty = true
			return nil
		}
		if f.focus == aclFieldTcpState {
			f.tcpStateIdx = (f.tcpStateIdx + 1) % len(aclTcpStateOrder)
			f.dirty = true
			return nil
		}
	}

	idx := f.inputIndex(f.focus)
	if idx >= 0 && idx < len(f.inputs) {
		prev := f.inputs[idx].Value()
		var cmd tea.Cmd
		f.inputs[idx], cmd = f.inputs[idx].Update(msg)
		if f.inputs[idx].Value() != prev {
			f.dirty = true
		}
		return cmd
	}
	return nil
}

func (f *aclForm) View() string {
	var b strings.Builder
	title := tr("アクセスリストルール追加 (ACL Add)")
	if f.editing {
		title = fmt.Sprintf(tr("アクセスリストルール編集 (ID: %s)"), f.targetID)
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	renderRow := func(field aclFormField, label, val string) {
		marker := "  "
		if f.focus == field {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-22s %s\n", marker, label+":", val)
	}

	passVal := "[ Pass (許可) ]"
	if !f.pass {
		passVal = "[ Discard (破棄) ]"
	}
	renderRow(aclFieldPass, tr("動作 (Action)"), passVal)

	statusVal := "[ Enable (有効) ]"
	if !f.enable {
		statusVal = "[ Disable (無効) ]"
	}
	renderRow(aclFieldEnable, tr("ステータス (Status)"), statusVal)

	renderRow(aclFieldPriority, tr("優先度 (Priority)"), f.inputs[0].View())
	renderRow(aclFieldMemo, tr("ルール説明 (Memo)"), f.inputs[1].View())
	renderRow(aclFieldProtocol, tr("プロトコル (Protocol)"), "["+aclProtocolOrder[f.protocolIdx].Name+"]")
	renderRow(aclFieldSrcIP, tr("送信元 IP (Src IP)"), f.inputs[2].View())
	renderRow(aclFieldDstIP, tr("送信先 IP (Dst IP)"), f.inputs[3].View())
	renderRow(aclFieldSrcPort, tr("送信元 ポート (Src Port)"), f.inputs[4].View())
	renderRow(aclFieldDstPort, tr("送信先 ポート (Dst Port)"), f.inputs[5].View())
	renderRow(aclFieldTcpState, tr("TCP 状態 (TCP State)"), "["+aclTcpStateOrder[f.tcpStateIdx].Name+"]")
	renderRow(aclFieldSrcUser, tr("送信元 User (Src User)"), f.inputs[6].View())
	renderRow(aclFieldDstUser, tr("送信先 User (Dst User)"), f.inputs[7].View())
	renderRow(aclFieldSrcMAC, tr("送信元 MAC (Src MAC)"), f.inputs[8].View())
	renderRow(aclFieldDstMAC, tr("送信先 MAC (Dst MAC)"), f.inputs[9].View())

	prioValid := strings.TrimSpace(f.inputs[0].Value()) != ""
	memoValid := strings.TrimSpace(f.inputs[1].Value()) != ""
	canSave := prioValid && memoValid

	b.WriteString("\n")
	saveMarker := "  "
	if f.focus == aclFieldSave {
		saveMarker = "> "
	}

	if canSave {
		if f.focus == aclFieldSave {
			b.WriteString(saveMarker + saveKeyStyle.Render(" [ Save ] ") + "\n")
		} else {
			b.WriteString(saveMarker + inactiveTabStyle.Render(" [ Save ] ") + "\n")
		}
		b.WriteString("\n" + renderHelp(
			"Tab/↑↓", tr("項目移動"),
			"←/→", tr("選択肢変更"),
			"Enter", tr("保存 (Save)"),
			"Esc", tr("キャンセル"),
		))
	} else {
		b.WriteString(saveMarker + dimStyle.Render("[ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp(
			"Tab/↑↓", tr("項目移動"),
			"Esc", tr("キャンセル"),
		))
	}
	return b.String()
}
