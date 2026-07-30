package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

// bridgeState is the Local Bridge management screen (app_specs.md 5.8 /
// 8.1). Like listenerState, local bridges are server-wide, so this is
// reached directly from the Dashboard rather than scoped to a single hub.
type bridgeState struct {
	profile config.Profile
	devices vpncmd.Table
	bridges vpncmd.Table
	cursor  int
	loading bool
	err     error
}

func (d bridgeState) currentHubName() (string, bool) {
	if d.cursor < 0 || d.cursor >= len(d.bridges.Rows) {
		return "", false
	}
	return d.bridges.NameOf(d.bridges.Rows[d.cursor]), true
}

func (d bridgeState) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("ローカルブリッジ管理")) + "\n\n")

	switch {
	case d.loading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.err != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.err.Error()) + "\n")
	default:
		b.WriteString(headerStyle.Render(tr("利用可能な物理デバイス")) + "\n")
		b.WriteString(renderTable(d.devices, -1))
		b.WriteString("\n" + headerStyle.Render(tr("ローカルブリッジ一覧")) + "\n")
		b.WriteString(renderTable(d.bridges, d.cursor))
	}

	b.WriteString("\n" + dimStyle.Render(tr("↑/↓:選択  a:追加  d:削除  r:更新  Esc:戻る  q:終了")))
	return b.String()
}

// --- messages ---

type bridgesLoadedMsg struct {
	devices vpncmd.Table
	bridges vpncmd.Table
	err     error
}

type bridgeActionResultMsg struct {
	action  string
	hubName string
	err     error
}

// --- commands ---

func (m Model) fetchBridges(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		devices, err := client.BridgeDeviceList(ctx, target)
		if err != nil {
			return bridgesLoadedMsg{err: err}
		}
		bridges, err := client.BridgeList(ctx, target)
		if err != nil {
			return bridgesLoadedMsg{err: err}
		}
		return bridgesLoadedMsg{devices: devices, bridges: bridges}
	}
}

func (m Model) createBridge(p config.Profile, hubName, deviceName string, tap bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.BridgeCreate(ctx, target, hubName, deviceName, tap)
		return bridgeActionResultMsg{action: tr("作成"), hubName: hubName, err: err}
	}
}

func (m Model) deleteBridge(p config.Profile, hubName string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.BridgeDelete(ctx, target, hubName)
		return bridgeActionResultMsg{action: tr("削除"), hubName: hubName, err: err}
	}
}

// --- key handling ---

func (m Model) handleBridgeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace":
		m.screen = screenDashboard

	case "r":
		m.bridge.loading = true
		return m, m.fetchBridges(m.bridge.profile)

	case "up", "k":
		if m.bridge.cursor > 0 {
			m.bridge.cursor--
		}

	case "down", "j":
		if m.bridge.cursor < len(m.bridge.bridges.Rows)-1 {
			m.bridge.cursor++
		}

	case "a":
		m.bridgeForm.Reset()
		m.screen = screenBridgeForm
		m.status = ""

	case "d":
		if name, ok := m.bridge.currentHubName(); ok {
			m.confirm.Show(confirmDeleteBridge, name, fmt.Sprintf(tr("Hub %q のローカルブリッジを削除しますか?"), name))
		}
	}
	return m, nil
}
