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

// clientDashboardState is the top screen for a VPN Client (/CLIENT) profile:
// it lists connection accounts, since Client mode has no Hub concept to
// drill into (unlike Server/Bridge's dashboardState + hubDetailState).
type clientDashboardState struct {
	profile config.Profile
	table   vpncmd.Table
	cursor  int
	loading bool
	err     error
}

func (d clientDashboardState) currentAccountName() (string, bool) {
	if d.cursor < 0 || d.cursor >= len(d.table.Rows) {
		return "", false
	}
	return d.table.NameOf(d.table.Rows[d.cursor]), true
}

func (d clientDashboardState) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render(fmt.Sprintf("%s (%s) - VPN Client", d.profile.Name, d.profile.Address())))

	switch {
	case d.loading:
		b.WriteString(tr("接続確認中...") + "\n")
	case d.err != nil:
		b.WriteString(errorStyle.Render(tr("接続エラー: ")+d.err.Error()) + "\n")
	default:
		b.WriteString(headerStyle.Render(tr("接続一覧")) + "\n")
		b.WriteString(renderTable(d.table, d.cursor))
	}

	b.WriteString("\n" + renderHelp(
		"↑/↓", tr("選択"),
		"c", tr("作成"),
		"d", tr("削除"),
		"o", tr("接続"),
		"f", tr("切断"),
		"u", tr("ユーザー名変更"),
		"p", tr("パスワード再設定"),
		"r", tr("更新"),
		"Esc", tr("戻る"),
		"q", tr("終了"),
	))
	return b.String()
}

// --- messages ---

type accountsLoadedMsg struct {
	profileName string
	table       vpncmd.Table
	err         error
}

type accountCreateResultMsg struct {
	name string
	err  error
}

type accountDeleteResultMsg struct {
	name string
	err  error
}

// accountActionResultMsg reports the outcome of a connect/disconnect/
// password-reset action against an existing account.
type accountActionResultMsg struct {
	action string
	name   string
	err    error
}

// --- commands ---

func (m Model) fetchAccounts(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	name := p.Name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.AccountList(ctx, target)
		return accountsLoadedMsg{profileName: name, table: table, err: err}
	}
}

func (m Model) createAccount(p config.Profile, name string, opts vpncmd.AccountCreateOptions, authType vpncmd.AccountAuthType, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := client.AccountCreate(ctx, target, name, opts); err != nil {
			return accountCreateResultMsg{name: name, err: err}
		}
		var err error
		switch authType {
		case vpncmd.AccountAuthPassword:
			err = client.AccountPasswordSet(ctx, target, name, password)
		case vpncmd.AccountAuthAnonymous:
			err = client.AccountAnonymousSet(ctx, target, name)
		}
		return accountCreateResultMsg{name: name, err: err}
	}
}

func (m Model) deleteAccount(p config.Profile, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.AccountDelete(ctx, target, name)
		return accountDeleteResultMsg{name: name, err: err}
	}
}

func (m Model) setAccountConnected(p config.Profile, name string, connect bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	action := tr("切断")
	if connect {
		action = tr("接続")
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if connect {
			err = client.AccountConnect(ctx, target, name)
		} else {
			err = client.AccountDisconnect(ctx, target, name)
		}
		return accountActionResultMsg{action: action, name: name, err: err}
	}
}

func (m Model) setAccountPassword(p config.Profile, name, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.AccountPasswordSet(ctx, target, name, password)
		return accountActionResultMsg{action: tr("パスワード再設定"), name: name, err: err}
	}
}

func (m Model) setAccountUsername(p config.Profile, name, username string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.AccountUsernameSet(ctx, target, name, username)
		return accountActionResultMsg{action: tr("ユーザー名変更"), name: name, err: err}
	}
}

// --- key handling ---

func (m Model) handleClientDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		m.screen = screenProfileList

	case "r":
		m.clientDashboard.loading = true
		return m, m.fetchAccounts(m.clientDashboard.profile)

	case "up", "k":
		if m.clientDashboard.cursor > 0 {
			m.clientDashboard.cursor--
		}

	case "down", "j":
		if m.clientDashboard.cursor < len(m.clientDashboard.table.Rows)-1 {
			m.clientDashboard.cursor++
		}

	case "c", "C", "a", "A":
		m.accountForm.Reset()
		m.screen = screenAccountForm
		m.status = ""

	case "d":
		if name, ok := m.clientDashboard.currentAccountName(); ok {
			m.confirm.Show(confirmDeleteAccount, name, fmt.Sprintf(tr("接続 %q を削除しますか?"), name))
		}

	case "o":
		if name, ok := m.clientDashboard.currentAccountName(); ok {
			m.status = fmt.Sprintf(tr("接続 %q を開始しています..."), name)
			m.statusErr = false
			return m, m.setAccountConnected(m.clientDashboard.profile, name, true)
		}

	case "f":
		if name, ok := m.clientDashboard.currentAccountName(); ok {
			m.status = fmt.Sprintf(tr("接続 %q を切断しています..."), name)
			m.statusErr = false
			return m, m.setAccountConnected(m.clientDashboard.profile, name, false)
		}

	case "u":
		if name, ok := m.clientDashboard.currentAccountName(); ok {
			m.prompt.Show(promptAccountUsername, name, fmt.Sprintf(tr("接続 %q の新しいユーザー名"), name), tr("ユーザー名"), false)
		}

	case "p":
		if name, ok := m.clientDashboard.currentAccountName(); ok {
			m.prompt.Show(promptAccountPassword, name, fmt.Sprintf(tr("接続 %q の新しいパスワード"), name), tr("新しいパスワード"), true)
		}
	}
	return m, nil
}

func (m Model) handleAccountFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenClientDashboard
		return m, nil

	case "enter":
		name, opts, authType, password, err := m.accountForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("接続 %q を作成しています..."), name)
		m.statusErr = false
		return m, m.createAccount(m.clientDashboard.profile, name, opts, authType, password)
	}

	cmd := m.accountForm.Update(msg)
	return m, cmd
}
