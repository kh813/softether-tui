package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

// listenerState is the Listener management screen (app_specs.md 5.4 / 8.1).
// Listeners are server-wide, so unlike the Hub detail tabs this is reached
// directly from the Dashboard rather than scoped to a single hub.
type listenerState struct {
	profile config.Profile
	table   vpncmd.Table
	cursor  int
	loading bool
	err     error
}

func (d listenerState) currentPort() (string, bool) {
	if d.cursor < 0 || d.cursor >= len(d.table.Rows) {
		return "", false
	}
	return d.table.NameOf(d.table.Rows[d.cursor]), true
}

func (d listenerState) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("リスナー管理")) + "\n\n")

	switch {
	case d.loading:
		b.WriteString(tr("読み込み中...") + "\n")
	case d.err != nil:
		b.WriteString(errorStyle.Render(tr("エラー: ")+d.err.Error()) + "\n")
	default:
		b.WriteString(renderTable(d.table, d.cursor))
	}

	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("選択"),
		"n", tr("作成"),
		"d", tr("削除"),
		"o", tr("有効化"),
		"f", tr("無効化"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
	return b.String()
}

// --- messages ---

type listenersLoadedMsg struct {
	table vpncmd.Table
	err   error
}

type listenerActionResultMsg struct {
	action string
	port   string
	err    error
}

// --- commands ---

func (m Model) fetchListeners(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.ListenerList(ctx, target)
		return listenersLoadedMsg{table: table, err: err}
	}
}

func (m Model) createListener(p config.Profile, port int) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.ListenerCreate(ctx, target, port)
		return listenerActionResultMsg{action: tr("作成"), port: strconv.Itoa(port), err: err}
	}
}

func (m Model) deleteListener(p config.Profile, port string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		n, err := strconv.Atoi(port)
		if err != nil {
			return listenerActionResultMsg{action: tr("削除"), port: port, err: fmt.Errorf(tr("ポート番号が不正です: %q"), port)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err = client.ListenerDelete(ctx, target, n)
		return listenerActionResultMsg{action: tr("削除"), port: port, err: err}
	}
}

func (m Model) setListenerEnabled(p config.Profile, port string, enabled bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	action := tr("無効化")
	if enabled {
		action = tr("有効化")
	}
	return func() tea.Msg {
		n, err := strconv.Atoi(port)
		if err != nil {
			return listenerActionResultMsg{action: action, port: port, err: fmt.Errorf(tr("ポート番号が不正です: %q"), port)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if enabled {
			err = client.ListenerEnable(ctx, target, n)
		} else {
			err = client.ListenerDisable(ctx, target, n)
		}
		return listenerActionResultMsg{action: action, port: port, err: err}
	}
}

// --- key handling ---

func (m Model) handleListenerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace":
		m.screen = screenDashboard

	case "r":
		m.listener.loading = true
		return m, m.fetchListeners(m.listener.profile)

	case "up", "k":
		if m.listener.cursor > 0 {
			m.listener.cursor--
		}

	case "down", "j":
		if m.listener.cursor < len(m.listener.table.Rows)-1 {
			m.listener.cursor++
		}

	case "n":
		m.prompt.Show(promptListenerCreate, "", tr("作成するポート番号"), tr("例: 1194"), false)

	case "d":
		if port, ok := m.listener.currentPort(); ok {
			m.confirm.Show(confirmDeleteListener, port, fmt.Sprintf(tr("ポート %q のリスナーを削除しますか?"), port))
		}

	case "o":
		if port, ok := m.listener.currentPort(); ok {
			m.confirm.Show(confirmEnableListener, port, fmt.Sprintf(tr("ポート %q のリスナーを有効化しますか?"), port))
		}

	case "f":
		if port, ok := m.listener.currentPort(); ok {
			m.confirm.Show(confirmDisableListener, port, fmt.Sprintf(tr("ポート %q のリスナーを無効化しますか?"), port))
		}
	}
	return m, nil
}
