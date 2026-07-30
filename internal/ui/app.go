// Package ui implements the Bubble Tea screens for softether-tui.
package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

type screen int

const (
	screenProfileList screen = iota
	screenProfileForm
	screenDashboard
	screenHubDetail
	screenHubForm
	screenUserForm
	screenGroupForm
	screenListener
	screenBridge
	screenBridgeForm
	screenClientDashboard
	screenAccountForm
)

// Model is the root Bubble Tea model: it routes key input and messages to
// whichever screen is active and owns the data shared across screens
// (profiles, the vpncmd client, in-flight status).
type Model struct {
	store  *config.Store
	client *vpncmd.Client

	version string

	profiles    []config.Profile
	cursor      int
	testResults map[string]error

	screen      screen
	confirm     confirmDialog
	prompt      prompt
	form        *profileForm
	hubForm     *hubForm
	userForm    *userForm
	groupForm   *groupForm
	bridgeForm  *bridgeForm
	accountForm *accountForm

	dashboard       dashboardState
	hubDetail       hubDetailState
	listener        listenerState
	bridge          bridgeState
	clientDashboard clientDashboardState

	status    string
	statusErr bool

	sessionPasswords        map[string]string
	initialPasswordPrompted map[string]bool

	quitting bool
	width    int
}

func New(store *config.Store, client *vpncmd.Client, version string) Model {
	return Model{
		store:                   store,
		client:                  client,
		version:                 version,
		form:                    newProfileForm(),
		hubForm:                 newHubForm(),
		userForm:                newUserForm(),
		groupForm:               newGroupForm(),
		bridgeForm:              newBridgeForm(),
		accountForm:             newAccountForm(),
		testResults:             map[string]error{},
		sessionPasswords:        map[string]string{},
		initialPasswordPrompted: map[string]bool{},
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadProfiles
}

// --- messages ---

type profilesLoadedMsg struct {
	profiles []config.Profile
	err      error
}

type profilesSavedMsg struct{ err error }

type testResultMsg struct {
	name string
	err  error
}

type serverInfoMsg struct {
	profileName string
	info        vpncmd.KeyValue
	status      vpncmd.KeyValue
	hubs        vpncmd.Table
	err         error
}

type serverPasswordSetResultMsg struct {
	profileName string
	err         error
}

type hubDetailMsg struct {
	hubName string
	info    vpncmd.KeyValue
	err     error
}

type hubCreateResultMsg struct {
	name string
	err  error
}

type hubDeleteResultMsg struct {
	name string
	err  error
}

type hubOnlineResultMsg struct {
	hubName string
	online  bool
	err     error
}

type usersLoadedMsg struct {
	hubName string
	table   vpncmd.Table
	err     error
}

type groupsLoadedMsg struct {
	hubName string
	table   vpncmd.Table
	err     error
}

type userCreateResultMsg struct {
	name string
	err  error
}

type userDeleteResultMsg struct {
	name string
	err  error
}

// userActionResultMsg reports the outcome of a prompt-driven per-user action
// (password reset, group change, expiration date).
type userActionResultMsg struct {
	action string
	name   string
	err    error
}

type groupCreateResultMsg struct {
	name string
	err  error
}

type groupDeleteResultMsg struct {
	name string
	err  error
}

type sessionsLoadedMsg struct {
	hubName string
	table   vpncmd.Table
	err     error
}

type sessionDisconnectResultMsg struct {
	name string
	err  error
}

type logLoadedMsg struct {
	hubName string
	info    vpncmd.KeyValue
	err     error
}

// sessionTickMsg drives the Sessions tab's auto-refresh. gen guards against
// a stale tick chain (from a tab/hub the user has since left) rescheduling
// itself forever.
type sessionTickMsg struct {
	hubName string
	gen     int
}

// --- commands ---

func (m Model) loadProfiles() tea.Msg {
	profiles, err := m.store.Load()
	return profilesLoadedMsg{profiles: profiles, err: err}
}

func (m Model) saveProfiles() tea.Cmd {
	store := m.store
	profiles := m.profiles
	return func() tea.Msg {
		return profilesSavedMsg{err: store.Save(profiles)}
	}
}

func (m Model) testConnection(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	name := p.Name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if p.Mode == config.ModeClient {
			// VPN Client mode has no ServerInfoGet-equivalent; AccountList
			// is the cheapest call that proves the client service is reachable.
			_, err = client.AccountList(ctx, target)
		} else {
			_, err = client.ServerInfo(ctx, target)
		}
		return testResultMsg{name: name, err: err}
	}
}

func (m Model) fetchServerInfo(p config.Profile) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	name := p.Name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		info, err := client.ServerInfo(ctx, target)
		if err != nil {
			return serverInfoMsg{profileName: name, err: err}
		}
		status, err := client.ServerStatus(ctx, target)
		if err != nil {
			return serverInfoMsg{profileName: name, err: err}
		}
		hubs, err := client.HubList(ctx, target)
		if err != nil {
			return serverInfoMsg{profileName: name, err: err}
		}
		return serverInfoMsg{profileName: name, info: info, status: status, hubs: hubs}
	}
}

func (m Model) fetchHubDetail(p config.Profile, hubName string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		info, err := client.HubGet(ctx, target, hubName)
		return hubDetailMsg{hubName: hubName, info: info, err: err}
	}
}

func (m Model) setServerPassword(p config.Profile, newPassword string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.ServerPasswordSet(ctx, target, newPassword)
		return serverPasswordSetResultMsg{profileName: p.Name, err: err}
	}
}

func (m Model) createHub(p config.Profile, name, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.HubCreate(ctx, target, name, password)
		return hubCreateResultMsg{name: name, err: err}
	}
}

func (m Model) deleteHub(p config.Profile, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.HubDelete(ctx, target, name)
		return hubDeleteResultMsg{name: name, err: err}
	}
}

func (m Model) setHubOnline(p config.Profile, hubName string, online bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.HubSetOnline(ctx, target, hubName, online)
		return hubOnlineResultMsg{hubName: hubName, online: online, err: err}
	}
}

func (m Model) fetchUsers(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.UserList(ctx, target)
		return usersLoadedMsg{hubName: hub, table: table, err: err}
	}
}

func (m Model) fetchGroups(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.GroupList(ctx, target)
		return groupsLoadedMsg{hubName: hub, table: table, err: err}
	}
}

func (m Model) createUser(p config.Profile, hub, name string, opts vpncmd.UserCreateOptions, authType vpncmd.UserAuthType, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := client.UserCreate(ctx, target, name, opts); err != nil {
			return userCreateResultMsg{name: name, err: err}
		}
		var err error
		switch authType {
		case vpncmd.UserAuthPassword:
			err = client.UserPasswordSet(ctx, target, name, password)
		case vpncmd.UserAuthAnonymous:
			err = client.UserAnonymousSet(ctx, target, name)
		case vpncmd.UserAuthRadius:
			err = client.UserRadiusSet(ctx, target, name)
		}
		return userCreateResultMsg{name: name, err: err}
	}
}

func (m Model) deleteUser(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.UserDelete(ctx, target, name)
		return userDeleteResultMsg{name: name, err: err}
	}
}

func (m Model) setUserPassword(p config.Profile, hub, name, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.UserPasswordSet(ctx, target, name, password)
		return userActionResultMsg{action: tr("パスワード再設定"), name: name, err: err}
	}
}

func (m Model) setUserGroup(p config.Profile, hub, name, group string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.UserSetGroup(ctx, target, name, group)
		return userActionResultMsg{action: tr("グループ変更"), name: name, err: err}
	}
}

func (m Model) setUserExpires(p config.Profile, hub, name string, expires time.Time) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.UserExpiresSet(ctx, target, name, expires)
		return userActionResultMsg{action: tr("有効期限設定"), name: name, err: err}
	}
}

func (m Model) createGroup(p config.Profile, hub, name string, opts vpncmd.GroupCreateOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := client.GroupCreate(ctx, target, name, opts)
		return groupCreateResultMsg{name: name, err: err}
	}
}

func (m Model) deleteGroup(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.GroupDelete(ctx, target, name)
		return groupDeleteResultMsg{name: name, err: err}
	}
}

func (m Model) fetchSessions(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		table, err := client.SessionList(ctx, target)
		return sessionsLoadedMsg{hubName: hub, table: table, err: err}
	}
}

func (m Model) disconnectSession(p config.Profile, hub, name string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.SessionDisconnect(ctx, target, name)
		return sessionDisconnectResultMsg{name: name, err: err}
	}
}

func (m Model) fetchLog(p config.Profile, hub string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		info, err := client.LogGet(ctx, target)
		return logLoadedMsg{hubName: hub, info: info, err: err}
	}
}

func sessionTick(hubName string, gen int, interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return sessionTickMsg{hubName: hubName, gen: gen}
	})
}

// startSessionAutoRefresh fetches the session list immediately and starts a
// new auto-refresh tick chain, invalidating any previous chain via sessionGen.
func (m Model) startSessionAutoRefresh() (tea.Model, tea.Cmd) {
	m.hubDetail.sessionsLoading = true
	m.hubDetail.sessionGen++
	gen := m.hubDetail.sessionGen
	return m, tea.Batch(
		m.fetchSessions(m.hubDetail.profile, m.hubDetail.hubName),
		sessionTick(m.hubDetail.hubName, gen, m.hubDetail.refreshInterval),
	)
}

func (m Model) targetFromProfile(p config.Profile) vpncmd.Target {
	pw := m.passwordFromEnvOrSession(p)
	return vpncmd.Target{
		Host:     p.Host,
		Port:     p.Port,
		Mode:     p.Mode,
		Hub:      p.Hub,
		Password: pw,
	}
}

func (m Model) passwordFromEnvOrSession(p config.Profile) string {
	if pw, ok := m.sessionPasswords[p.Name]; ok {
		return pw
	}
	if p.PasswordEnv != "" {
		return os.Getenv(p.PasswordEnv)
	}
	return ""
}

// --- update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case profilesLoadedMsg:
		if msg.err != nil {
			m.status = tr("プロファイル読込エラー: ") + msg.err.Error()
			m.statusErr = true
		} else {
			m.profiles = msg.profiles
		}
		return m, nil

	case profilesSavedMsg:
		if msg.err != nil {
			m.status = tr("保存エラー: ") + msg.err.Error()
			m.statusErr = true
		}
		return m, nil

	case testResultMsg:
		m.testResults[msg.name] = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("%s: 接続テスト失敗 (%s)"), msg.name, msg.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf(tr("%s: 接続テスト成功"), msg.name)
			m.statusErr = false
		}
		return m, nil

	case serverInfoMsg:
		p := m.dashboard.profile
		currentPw := m.passwordFromEnvOrSession(p)
		if msg.err != nil && (strings.Contains(msg.err.Error(), "Access has been denied") || strings.Contains(msg.err.Error(), "exit status 1")) {
			if currentPw == "" {
				// Prompt user to enter admin password
				m.dashboard.loading = false
				m.dashboard.err = msg.err
				m.prompt.Show(promptConnectPassword, p.Name, fmt.Sprintf(tr("管理者パスワードを入力してください (%s)"), p.Name), tr("パスワード"), true)
				return m, nil
			}
		}

		m.dashboard.loading = false
		m.dashboard.err = msg.err
		m.dashboard.info = msg.info
		m.dashboard.status = msg.status
		m.dashboard.hubs = msg.hubs
		if m.dashboard.hubCursor >= len(msg.hubs.Rows) {
			m.dashboard.hubCursor = len(msg.hubs.Rows) - 1
		}
		if m.dashboard.hubCursor < 0 {
			m.dashboard.hubCursor = 0
		}
		m.testResults[msg.profileName] = msg.err

		// Check if initial password setup is needed
		if msg.err == nil && p.Mode != config.ModeClient {
			if !m.initialPasswordPrompted[p.Name] {
				if currentPw == "" {
					m.initialPasswordPrompted[p.Name] = true
					m.prompt.Show(promptInitialPassword, p.Name, fmt.Sprintf(tr("初回接続: 新しい管理者パスワードを設定してください (%s)"), p.Name), tr("新しいパスワード (空欄で変更なし)"), true)
				}
			}
		}

		return m, nil

	case serverPasswordSetResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("管理者パスワードの設定に失敗しました: %s"), msg.err.Error())
			m.statusErr = true
		} else {
			m.status = tr("管理者パスワードを設定しました")
			m.statusErr = false
		}
		return m, nil

	case hubDetailMsg:
		if m.screen == screenHubDetail && m.hubDetail.hubName == msg.hubName {
			m.hubDetail.loading = false
			m.hubDetail.info = msg.info
			m.hubDetail.err = msg.err
		}
		return m, nil

	case hubCreateResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("Hub %q の作成に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q を作成しました"), msg.name)
		m.statusErr = false
		m.screen = screenDashboard
		m.dashboard.loading = true
		return m, m.fetchServerInfo(m.dashboard.profile)

	case hubDeleteResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("Hub %q の削除に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q を削除しました"), msg.name)
		m.statusErr = false
		m.dashboard.loading = true
		return m, m.fetchServerInfo(m.dashboard.profile)

	case hubOnlineResultMsg:
		label := tr("オフライン化")
		if msg.online {
			label = tr("オンライン化")
		}
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("Hub %q の%sに失敗しました: %s"), msg.hubName, label, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q を%sしました"), msg.hubName, label)
		m.statusErr = false
		if m.screen == screenHubDetail && m.hubDetail.hubName == msg.hubName {
			return m, m.fetchHubDetail(m.hubDetail.profile, msg.hubName)
		}
		return m, nil

	case usersLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.usersLoading = false
			m.hubDetail.usersLoaded = true
			m.hubDetail.users = msg.table
			m.hubDetail.usersErr = msg.err
			if m.hubDetail.userCursor >= len(msg.table.Rows) {
				m.hubDetail.userCursor = len(msg.table.Rows) - 1
			}
			if m.hubDetail.userCursor < 0 {
				m.hubDetail.userCursor = 0
			}
		}
		return m, nil

	case groupsLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.groupsLoading = false
			m.hubDetail.groupsLoaded = true
			m.hubDetail.groups = msg.table
			m.hubDetail.groupsErr = msg.err
			if m.hubDetail.groupCursor >= len(msg.table.Rows) {
				m.hubDetail.groupCursor = len(msg.table.Rows) - 1
			}
			if m.hubDetail.groupCursor < 0 {
				m.hubDetail.groupCursor = 0
			}
		}
		return m, nil

	case userCreateResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("ユーザー %q の作成に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ユーザー %q を作成しました"), msg.name)
		m.statusErr = false
		m.screen = screenHubDetail
		return m, m.fetchUsers(m.hubDetail.profile, m.hubDetail.hubName)

	case userDeleteResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("ユーザー %q の削除に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ユーザー %q を削除しました"), msg.name)
		m.statusErr = false
		return m, m.fetchUsers(m.hubDetail.profile, m.hubDetail.hubName)

	case userActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("%s (%s) に失敗しました: %s"), msg.action, msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("%s (%s) が完了しました"), msg.action, msg.name)
		m.statusErr = false
		return m, m.fetchUsers(m.hubDetail.profile, m.hubDetail.hubName)

	case groupCreateResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("グループ %q の作成に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("グループ %q を作成しました"), msg.name)
		m.statusErr = false
		m.screen = screenHubDetail
		return m, m.fetchGroups(m.hubDetail.profile, m.hubDetail.hubName)

	case groupDeleteResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("グループ %q の削除に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("グループ %q を削除しました"), msg.name)
		m.statusErr = false
		return m, m.fetchGroups(m.hubDetail.profile, m.hubDetail.hubName)

	case sessionsLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.sessionsLoading = false
			m.hubDetail.sessionsLoaded = true
			m.hubDetail.sessions = msg.table
			m.hubDetail.sessionsErr = msg.err
			m.hubDetail.lastRefreshed = time.Now()
			if m.hubDetail.sessionCursor >= len(msg.table.Rows) {
				m.hubDetail.sessionCursor = len(msg.table.Rows) - 1
			}
			if m.hubDetail.sessionCursor < 0 {
				m.hubDetail.sessionCursor = 0
			}
		}
		return m, nil

	case sessionDisconnectResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("セッション %q の切断に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("セッション %q を切断しました"), msg.name)
		m.statusErr = false
		return m, m.fetchSessions(m.hubDetail.profile, m.hubDetail.hubName)

	case logLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.logLoading = false
			m.hubDetail.logLoaded = true
			m.hubDetail.logInfo = msg.info
			m.hubDetail.logErr = msg.err
		}
		return m, nil

	case sessionTickMsg:
		if m.screen == screenHubDetail && m.hubDetail.tab == hubTabSessions &&
			m.hubDetail.hubName == msg.hubName && m.hubDetail.sessionGen == msg.gen {
			return m, tea.Batch(
				m.fetchSessions(m.hubDetail.profile, m.hubDetail.hubName),
				sessionTick(msg.hubName, msg.gen, m.hubDetail.refreshInterval),
			)
		}
		return m, nil

	case listenersLoadedMsg:
		m.listener.loading = false
		m.listener.table = msg.table
		m.listener.err = msg.err
		if m.listener.cursor >= len(msg.table.Rows) {
			m.listener.cursor = len(msg.table.Rows) - 1
		}
		if m.listener.cursor < 0 {
			m.listener.cursor = 0
		}
		return m, nil

	case listenerActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("リスナー %s (%s) に失敗しました: %s"), msg.action, msg.port, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("リスナー %s (%s) が完了しました"), msg.action, msg.port)
		m.statusErr = false
		return m, m.fetchListeners(m.listener.profile)

	case secureNatLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.secureNatLoading = false
			m.hubDetail.secureNatLoaded = true
			m.hubDetail.secureNatStatus = msg.status
			m.hubDetail.secureNatHost = msg.host
			m.hubDetail.secureNatErr = msg.err
		}
		return m, nil

	case secureNatActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("SecureNAT %s に失敗しました: %s"), msg.action, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("SecureNAT %s が完了しました"), msg.action)
		m.statusErr = false
		return m, m.fetchSecureNAT(m.hubDetail.profile, m.hubDetail.hubName)

	case accessLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.accessLoading = false
			m.hubDetail.accessLoaded = true
			m.hubDetail.access = msg.table
			m.hubDetail.accessErr = msg.err
			if m.hubDetail.accessCursor >= len(msg.table.Rows) {
				m.hubDetail.accessCursor = len(msg.table.Rows) - 1
			}
			if m.hubDetail.accessCursor < 0 {
				m.hubDetail.accessCursor = 0
			}
		}
		return m, nil

	case accessActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("ルール %q の%sに失敗しました: %s"), msg.id, msg.action, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ルール %q を%sしました"), msg.id, msg.action)
		m.statusErr = false
		return m, m.fetchAccessList(m.hubDetail.profile, m.hubDetail.hubName)

	case cascadeLoadedMsg:
		if m.hubDetail.hubName == msg.hubName {
			m.hubDetail.cascadeLoading = false
			m.hubDetail.cascadeLoaded = true
			m.hubDetail.cascade = msg.table
			m.hubDetail.cascadeErr = msg.err
			if m.hubDetail.cascadeCursor >= len(msg.table.Rows) {
				m.hubDetail.cascadeCursor = len(msg.table.Rows) - 1
			}
			if m.hubDetail.cascadeCursor < 0 {
				m.hubDetail.cascadeCursor = 0
			}
		}
		return m, nil

	case cascadeActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("カスケード接続 %q の%sに失敗しました: %s"), msg.name, msg.action, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("カスケード接続 %q を%sしました"), msg.name, msg.action)
		m.statusErr = false
		return m, m.fetchCascade(m.hubDetail.profile, m.hubDetail.hubName)

	case bridgesLoadedMsg:
		m.bridge.loading = false
		m.bridge.devices = msg.devices
		m.bridge.bridges = msg.bridges
		m.bridge.err = msg.err
		if m.bridge.cursor >= len(msg.bridges.Rows) {
			m.bridge.cursor = len(msg.bridges.Rows) - 1
		}
		if m.bridge.cursor < 0 {
			m.bridge.cursor = 0
		}
		return m, nil

	case bridgeActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("ローカルブリッジ %s (%s) に失敗しました: %s"), msg.action, msg.hubName, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ローカルブリッジ %s (%s) が完了しました"), msg.action, msg.hubName)
		m.statusErr = false
		if m.screen == screenBridgeForm {
			m.screen = screenBridge
		}
		return m, m.fetchBridges(m.bridge.profile)

	case accountsLoadedMsg:
		m.clientDashboard.loading = false
		m.clientDashboard.table = msg.table
		m.clientDashboard.err = msg.err
		if m.clientDashboard.cursor >= len(msg.table.Rows) {
			m.clientDashboard.cursor = len(msg.table.Rows) - 1
		}
		if m.clientDashboard.cursor < 0 {
			m.clientDashboard.cursor = 0
		}
		m.testResults[msg.profileName] = msg.err
		return m, nil

	case accountCreateResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("接続 %q の作成に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("接続 %q を作成しました"), msg.name)
		m.statusErr = false
		m.screen = screenClientDashboard
		return m, m.fetchAccounts(m.clientDashboard.profile)

	case accountDeleteResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("接続 %q の削除に失敗しました: %s"), msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("接続 %q を削除しました"), msg.name)
		m.statusErr = false
		return m, m.fetchAccounts(m.clientDashboard.profile)

	case accountActionResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("%s (%s) に失敗しました: %s"), msg.action, msg.name, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("%s (%s) が完了しました"), msg.action, msg.name)
		m.statusErr = false
		return m, m.fetchAccounts(m.clientDashboard.profile)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	if m.prompt.active {
		switch msg.String() {
		case "esc":
			m.prompt.Hide()
			return m, nil
		case "enter":
			return m.submitPrompt()
		}
		cmd := m.prompt.Update(msg)
		return m, cmd
	}

	if m.confirm.active {
		switch msg.String() {
		case "y", "Y":
			kind, target := m.confirm.kind, m.confirm.target
			m.confirm.Hide()
			return m.applyConfirm(kind, target)
		case "n", "N", "esc":
			m.confirm.Hide()
		}
		return m, nil
	}

	switch m.screen {
	case screenProfileList:
		return m.handleProfileListKey(msg)
	case screenProfileForm:
		return m.handleFormKey(msg)
	case screenDashboard:
		return m.handleDashboardKey(msg)
	case screenHubDetail:
		return m.handleHubDetailKey(msg)
	case screenHubForm:
		return m.handleHubFormKey(msg)
	case screenUserForm:
		return m.handleUserFormKey(msg)
	case screenGroupForm:
		return m.handleGroupFormKey(msg)
	case screenListener:
		return m.handleListenerKey(msg)
	case screenBridge:
		return m.handleBridgeKey(msg)
	case screenBridgeForm:
		return m.handleBridgeFormKey(msg)
	case screenClientDashboard:
		return m.handleClientDashboardKey(msg)
	case screenAccountForm:
		return m.handleAccountFormKey(msg)
	}
	return m, nil
}

func (m Model) submitPrompt() (tea.Model, tea.Cmd) {
	kind, target, value := m.prompt.kind, m.prompt.target, m.prompt.input.Value()
	m.prompt.Hide()

	profile := m.hubDetail.profile
	hub := m.hubDetail.hubName

	switch kind {
	case promptUserPassword:
		m.status = fmt.Sprintf(tr("%s のパスワードを再設定しています..."), target)
		m.statusErr = false
		return m, m.setUserPassword(profile, hub, target, value)

	case promptUserGroup:
		m.status = fmt.Sprintf(tr("%s のグループを変更しています..."), target)
		m.statusErr = false
		return m, m.setUserGroup(profile, hub, target, value)

	case promptUserExpires:
		expires, err := time.Parse("2006/01/02", strings.TrimSpace(value))
		if err != nil {
			m.status = tr("有効期限は YYYY/MM/DD 形式で入力してください")
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("%s の有効期限を設定しています..."), target)
		m.statusErr = false
		return m, m.setUserExpires(profile, hub, target, expires)

	case promptListenerCreate:
		port, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || port <= 0 || port > 65535 {
			m.status = fmt.Sprintf(tr("ポート番号が不正です: %q"), value)
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ポート %d のリスナーを作成しています..."), port)
		m.statusErr = false
		return m, m.createListener(m.listener.profile, port)

	case promptAccountPassword:
		m.status = fmt.Sprintf(tr("%s のパスワードを再設定しています..."), target)
		m.statusErr = false
		return m, m.setAccountPassword(m.clientDashboard.profile, target, value)

	case promptConnectPassword:
		m.sessionPasswords[target] = value
		if p, ok := m.currentProfile(); ok && p.Name == target {
			m.dashboard = dashboardState{profile: p, loading: true}
			m.screen = screenDashboard
			return m, m.fetchServerInfo(p)
		}

	case promptInitialPassword:
		if strings.TrimSpace(value) != "" {
			if p, ok := m.currentProfile(); ok && p.Name == target {
				m.status = fmt.Sprintf(tr("管理者パスワードを設定しています (%s)..."), target)
				m.statusErr = false
				m.sessionPasswords[target] = value
				return m, m.setServerPassword(p, value)
			}
		}

	case promptSecureNatHostIP:
		parts := strings.Split(strings.TrimSpace(value), "/")
		ip := parts[0]
		mask := "255.255.255.0"
		if len(parts) > 1 {
			mask = parts[1]
		}
		if ip != "" {
			m.status = fmt.Sprintf(tr("Hub %q の仮想ホスト IP を変更しています..."), hub)
			m.statusErr = false
			return m, m.setSecureNatHost(profile, hub, ip, mask)
		}

	case promptDhcpStart:
		parts := strings.Split(strings.TrimSpace(value), "-")
		startIp := parts[0]
		endIp := startIp
		if len(parts) > 1 {
			endIp = parts[1]
		}
		if startIp != "" {
			m.status = fmt.Sprintf(tr("Hub %q の DHCP 範囲を変更しています..."), hub)
			m.statusErr = false
			return m, m.setDhcpRange(profile, hub, startIp, endIp)
		}
	}
	return m, nil
}

func (m Model) applyConfirm(kind confirmKind, target string) (tea.Model, tea.Cmd) {
	switch kind {
	case confirmDeleteProfile:
		m.profiles = config.Remove(m.profiles, target)
		if m.cursor >= len(m.profiles) {
			m.cursor = len(m.profiles) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		delete(m.testResults, target)
		m.status = fmt.Sprintf(tr("プロファイル %q を削除しました"), target)
		m.statusErr = false
		return m, m.saveProfiles()

	case confirmDeleteHub:
		m.status = fmt.Sprintf(tr("Hub %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteHub(m.dashboard.profile, target)

	case confirmDeleteUser:
		m.status = fmt.Sprintf(tr("ユーザー %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteUser(m.hubDetail.profile, m.hubDetail.hubName, target)

	case confirmDeleteGroup:
		m.status = fmt.Sprintf(tr("グループ %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteGroup(m.hubDetail.profile, m.hubDetail.hubName, target)

	case confirmDisconnectSession:
		m.status = fmt.Sprintf(tr("セッション %q を切断しています..."), target)
		m.statusErr = false
		return m, m.disconnectSession(m.hubDetail.profile, m.hubDetail.hubName, target)

	case confirmDeleteListener:
		m.status = fmt.Sprintf(tr("リスナー %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteListener(m.listener.profile, target)

	case confirmDeleteAccessRule:
		m.status = fmt.Sprintf(tr("アクセスリストルール %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteAccessRule(m.hubDetail.profile, m.hubDetail.hubName, target)

	case confirmDeleteCascade:
		m.status = fmt.Sprintf(tr("カスケード接続 %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteCascade(m.hubDetail.profile, m.hubDetail.hubName, target)

	case confirmDeleteBridge:
		m.status = fmt.Sprintf(tr("ローカルブリッジ %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteBridge(m.bridge.profile, target)

	case confirmDeleteAccount:
		m.status = fmt.Sprintf(tr("接続 %q を削除しています..."), target)
		m.statusErr = false
		return m, m.deleteAccount(m.clientDashboard.profile, target)
	}
	return m, nil
}

func (m Model) currentProfile() (config.Profile, bool) {
	if m.cursor < 0 || m.cursor >= len(m.profiles) {
		return config.Profile{}, false
	}
	return m.profiles[m.cursor], true
}

func (m Model) handleProfileListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}

	case "a":
		m.form.Reset()
		m.screen = screenProfileForm
		m.status = ""

	case "e":
		if p, ok := m.currentProfile(); ok {
			m.form.LoadProfile(p)
			m.screen = screenProfileForm
			m.status = ""
		}

	case "d":
		if p, ok := m.currentProfile(); ok {
			m.confirm.Show(confirmDeleteProfile, p.Name, fmt.Sprintf(tr("プロファイル %q を削除しますか?"), p.Name))
		}

	case "t":
		if p, ok := m.currentProfile(); ok {
			m.status = fmt.Sprintf(tr("%s: 接続テスト中..."), p.Name)
			m.statusErr = false
			return m, m.testConnection(p)
		}

	case "enter":
		if p, ok := m.currentProfile(); ok {
			m.status = ""
			if p.Mode == config.ModeClient {
				m.clientDashboard = clientDashboardState{profile: p, loading: true}
				m.screen = screenClientDashboard
				return m, m.fetchAccounts(p)
			}
			m.dashboard = dashboardState{profile: p, loading: true}
			m.screen = screenDashboard
			return m, m.fetchServerInfo(p)
		}
	}
	return m, nil
}

func (m Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenProfileList
		return m, nil

	case "enter":
		p, err := m.form.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		if m.form.editing && m.form.original != p.Name {
			m.profiles = config.Remove(m.profiles, m.form.original)
			delete(m.testResults, m.form.original)
		}
		m.profiles = config.Upsert(m.profiles, p)
		m.screen = screenProfileList
		m.status = fmt.Sprintf(tr("プロファイル %q を保存しました"), p.Name)
		m.statusErr = false
		return m, m.saveProfiles()
	}

	cmd := m.form.Update(msg)
	return m, cmd
}

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace":
		m.screen = screenProfileList

	case "r":
		m.dashboard.loading = true
		return m, m.fetchServerInfo(m.dashboard.profile)

	case "up", "k":
		if m.dashboard.hubCursor > 0 {
			m.dashboard.hubCursor--
		}

	case "down", "j":
		if m.dashboard.hubCursor < len(m.dashboard.hubs.Rows)-1 {
			m.dashboard.hubCursor++
		}

	case "a":
		m.hubForm.Reset()
		m.screen = screenHubForm
		m.status = ""

	case "d":
		if name, ok := m.currentHubName(); ok {
			m.confirm.Show(confirmDeleteHub, name, fmt.Sprintf(tr("Hub %q を削除しますか?"), name))
		}

	case "l":
		m.listener = listenerState{profile: m.dashboard.profile, loading: true}
		m.screen = screenListener
		m.status = ""
		return m, m.fetchListeners(m.dashboard.profile)

	case "b":
		m.bridge = bridgeState{profile: m.dashboard.profile, loading: true}
		m.screen = screenBridge
		m.status = ""
		return m, m.fetchBridges(m.dashboard.profile)

	case "enter":
		if name, ok := m.currentHubName(); ok {
			m.hubDetail = hubDetailState{profile: m.dashboard.profile, hubName: name, loading: true, refreshInterval: 5 * time.Second}
			m.screen = screenHubDetail
			m.status = ""
			return m, m.fetchHubDetail(m.dashboard.profile, name)
		}
	}
	return m, nil
}

func (m Model) currentHubName() (string, bool) {
	d := m.dashboard
	if d.hubCursor < 0 || d.hubCursor >= len(d.hubs.Rows) {
		return "", false
	}
	return d.hubs.NameOf(d.hubs.Rows[d.hubCursor]), true
}

func (m Model) handleHubDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.hubDetail.filtering {
		return m.handleHubUserFilterKey(msg)
	}

	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace":
		m.screen = screenDashboard
		return m, nil

	case "tab":
		m.hubDetail.tab = (m.hubDetail.tab + 1) % hubTabCount
		return m.loadHubTabIfNeeded()

	case "shift+tab":
		m.hubDetail.tab = (m.hubDetail.tab - 1 + hubTabCount) % hubTabCount
		return m.loadHubTabIfNeeded()

	case "r":
		return m.refreshCurrentHubTab()
	}

	switch m.hubDetail.tab {
	case hubTabOverview:
		return m.handleHubOverviewKey(msg)
	case hubTabUsers:
		return m.handleHubUsersKey(msg)
	case hubTabGroups:
		return m.handleHubGroupsKey(msg)
	case hubTabSessions:
		return m.handleHubSessionsKey(msg)
	case hubTabSecureNAT:
		return m.handleHubSecureNATKey(msg)
	case hubTabACL:
		return m.handleHubACLKey(msg)
	case hubTabCascade:
		return m.handleHubCascadeKey(msg)
	}
	return m, nil
}

// loadHubTabIfNeeded fetches a tab's data the first time it is switched to,
// so opening Hub detail doesn't eagerly fetch users/groups nobody looks at.
// Sessions is the exception: switching into it always (re)starts
// auto-refresh, since that is the point of the tab.
func (m Model) loadHubTabIfNeeded() (tea.Model, tea.Cmd) {
	switch m.hubDetail.tab {
	case hubTabUsers:
		if !m.hubDetail.usersLoaded && !m.hubDetail.usersLoading {
			m.hubDetail.usersLoading = true
			return m, m.fetchUsers(m.hubDetail.profile, m.hubDetail.hubName)
		}
	case hubTabGroups:
		if !m.hubDetail.groupsLoaded && !m.hubDetail.groupsLoading {
			m.hubDetail.groupsLoading = true
			return m, m.fetchGroups(m.hubDetail.profile, m.hubDetail.hubName)
		}
	case hubTabSessions:
		return m.startSessionAutoRefresh()
	case hubTabLog:
		if !m.hubDetail.logLoaded && !m.hubDetail.logLoading {
			m.hubDetail.logLoading = true
			return m, m.fetchLog(m.hubDetail.profile, m.hubDetail.hubName)
		}
	case hubTabSecureNAT:
		if !m.hubDetail.secureNatLoaded && !m.hubDetail.secureNatLoading {
			m.hubDetail.secureNatLoading = true
			return m, m.fetchSecureNAT(m.hubDetail.profile, m.hubDetail.hubName)
		}
	case hubTabACL:
		if !m.hubDetail.accessLoaded && !m.hubDetail.accessLoading {
			m.hubDetail.accessLoading = true
			return m, m.fetchAccessList(m.hubDetail.profile, m.hubDetail.hubName)
		}
	case hubTabCascade:
		if !m.hubDetail.cascadeLoaded && !m.hubDetail.cascadeLoading {
			m.hubDetail.cascadeLoading = true
			return m, m.fetchCascade(m.hubDetail.profile, m.hubDetail.hubName)
		}
	}
	return m, nil
}

func (m Model) refreshCurrentHubTab() (tea.Model, tea.Cmd) {
	switch m.hubDetail.tab {
	case hubTabOverview:
		m.hubDetail.loading = true
		return m, m.fetchHubDetail(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabUsers:
		m.hubDetail.usersLoading = true
		return m, m.fetchUsers(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabGroups:
		m.hubDetail.groupsLoading = true
		return m, m.fetchGroups(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabSessions:
		return m.startSessionAutoRefresh()
	case hubTabLog:
		m.hubDetail.logLoading = true
		return m, m.fetchLog(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabSecureNAT:
		m.hubDetail.secureNatLoading = true
		return m, m.fetchSecureNAT(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabACL:
		m.hubDetail.accessLoading = true
		return m, m.fetchAccessList(m.hubDetail.profile, m.hubDetail.hubName)
	case hubTabCascade:
		m.hubDetail.cascadeLoading = true
		return m, m.fetchCascade(m.hubDetail.profile, m.hubDetail.hubName)
	}
	return m, nil
}

func (m Model) handleHubSessionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hubDetail.sessionCursor > 0 {
			m.hubDetail.sessionCursor--
		}

	case "down", "j":
		if m.hubDetail.sessionCursor < len(m.hubDetail.sessions.Rows)-1 {
			m.hubDetail.sessionCursor++
		}

	case "x":
		if name, ok := m.hubDetail.currentSessionName(); ok {
			m.confirm.Show(confirmDisconnectSession, name, fmt.Sprintf(tr("セッション %q を切断しますか?"), name))
		}

	case "+", "=":
		m.hubDetail.refreshInterval += time.Second
		if m.hubDetail.refreshInterval > 60*time.Second {
			m.hubDetail.refreshInterval = 60 * time.Second
		}

	case "-", "_":
		m.hubDetail.refreshInterval -= time.Second
		if m.hubDetail.refreshInterval < 2*time.Second {
			m.hubDetail.refreshInterval = 2 * time.Second
		}
	}
	return m, nil
}

func (m Model) handleHubOverviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		m.status = fmt.Sprintf(tr("Hub %q をオンライン化しています..."), m.hubDetail.hubName)
		m.statusErr = false
		return m, m.setHubOnline(m.hubDetail.profile, m.hubDetail.hubName, true)

	case "f":
		m.status = fmt.Sprintf(tr("Hub %q をオフライン化しています..."), m.hubDetail.hubName)
		m.statusErr = false
		return m, m.setHubOnline(m.hubDetail.profile, m.hubDetail.hubName, false)
	}
	return m, nil
}

func (m Model) handleHubUsersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hubDetail.userCursor > 0 {
			m.hubDetail.userCursor--
		}

	case "down", "j":
		if m.hubDetail.userCursor < len(m.hubDetail.filteredUsers())-1 {
			m.hubDetail.userCursor++
		}

	case "/":
		m.hubDetail.filtering = true
		ti := textinput.New()
		ti.SetValue(m.hubDetail.userFilter)
		ti.Focus()
		m.hubDetail.filterInput = ti

	case "a":
		m.userForm.Reset()
		m.screen = screenUserForm
		m.status = ""

	case "d":
		if name, ok := m.hubDetail.currentUserName(); ok {
			m.confirm.Show(confirmDeleteUser, name, fmt.Sprintf(tr("ユーザー %q を削除しますか?"), name))
		}

	case "p":
		if name, ok := m.hubDetail.currentUserName(); ok {
			m.prompt.Show(promptUserPassword, name, fmt.Sprintf(tr("ユーザー %q の新しいパスワード"), name), tr("新しいパスワード"), true)
		}

	case "g":
		if name, ok := m.hubDetail.currentUserName(); ok {
			m.prompt.Show(promptUserGroup, name, fmt.Sprintf(tr("ユーザー %q のグループ"), name), tr("グループ名 (空でグループ解除)"), false)
		}

	case "e":
		if name, ok := m.hubDetail.currentUserName(); ok {
			m.prompt.Show(promptUserExpires, name, fmt.Sprintf(tr("ユーザー %q の有効期限"), name), "YYYY/MM/DD", false)
		}
	}
	return m, nil
}

func (m Model) handleHubUserFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.hubDetail.userFilter = m.hubDetail.filterInput.Value()
		m.hubDetail.filtering = false
		m.hubDetail.userCursor = 0
		return m, nil
	case "esc":
		m.hubDetail.filtering = false
		return m, nil
	}
	var cmd tea.Cmd
	m.hubDetail.filterInput, cmd = m.hubDetail.filterInput.Update(msg)
	return m, cmd
}

func (m Model) handleHubGroupsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hubDetail.groupCursor > 0 {
			m.hubDetail.groupCursor--
		}

	case "down", "j":
		if m.hubDetail.groupCursor < len(m.hubDetail.groups.Rows)-1 {
			m.hubDetail.groupCursor++
		}

	case "a":
		m.groupForm.Reset()
		m.screen = screenGroupForm
		m.status = ""

	case "d":
		if name, ok := m.hubDetail.currentGroupName(); ok {
			m.confirm.Show(confirmDeleteGroup, name, fmt.Sprintf(tr("グループ %q を削除しますか?"), name))
		}
	}
	return m, nil
}

func (m Model) handleHubSecureNATKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		m.status = fmt.Sprintf(tr("Hub %q の SecureNAT を有効化しています..."), m.hubDetail.hubName)
		m.statusErr = false
		return m, m.setSecureNatEnabled(m.hubDetail.profile, m.hubDetail.hubName, true)

	case "f":
		m.status = fmt.Sprintf(tr("Hub %q の SecureNAT を無効化しています..."), m.hubDetail.hubName)
		m.statusErr = false
		return m, m.setSecureNatEnabled(m.hubDetail.profile, m.hubDetail.hubName, false)

	case "i":
		m.prompt.Show(promptSecureNatHostIP, m.hubDetail.hubName, tr("SecureNAT 仮想ホスト IP アドレス設定"), tr("例: 192.168.30.1/255.255.255.0"), false)

	case "s":
		m.prompt.Show(promptDhcpStart, m.hubDetail.hubName, tr("DHCP サーバー配布 IP 範囲設定"), tr("例: 192.168.30.10-192.168.30.200"), false)
	}
	return m, nil
}

func (m Model) handleUserFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHubDetail
		return m, nil

	case "enter":
		name, opts, authType, password, err := m.userForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ユーザー %q を作成しています..."), name)
		m.statusErr = false
		return m, m.createUser(m.hubDetail.profile, m.hubDetail.hubName, name, opts, authType, password)
	}

	cmd := m.userForm.Update(msg)
	return m, cmd
}

func (m Model) handleGroupFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHubDetail
		return m, nil

	case "enter":
		name, opts, err := m.groupForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("グループ %q を作成しています..."), name)
		m.statusErr = false
		return m, m.createGroup(m.hubDetail.profile, m.hubDetail.hubName, name, opts)
	}

	cmd := m.groupForm.Update(msg)
	return m, cmd
}

func (m Model) handleBridgeFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenBridge
		return m, nil

	case "enter":
		hubName, deviceName, tap, err := m.bridgeForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("ローカルブリッジ (Hub %q) を作成しています..."), hubName)
		m.statusErr = false
		return m, m.createBridge(m.bridge.profile, hubName, deviceName, tap)
	}

	cmd := m.bridgeForm.Update(msg)
	return m, cmd
}

func (m Model) handleHubFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDashboard
		return m, nil

	case "enter":
		name, password, err := m.hubForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q を作成しています..."), name)
		m.statusErr = false
		return m, m.createHub(m.dashboard.profile, name, password)
	}

	cmd := m.hubForm.Update(msg)
	return m, cmd
}

// --- view ---

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var body string
	switch m.screen {
	case screenProfileList:
		body = m.viewProfileList()
	case screenProfileForm:
		body = m.form.View()
	case screenDashboard:
		body = m.dashboard.View()
	case screenHubDetail:
		body = m.hubDetail.View()
	case screenHubForm:
		body = m.hubForm.View()
	case screenUserForm:
		body = m.userForm.View()
	case screenGroupForm:
		body = m.groupForm.View()
	case screenListener:
		body = m.listener.View()
	case screenBridge:
		body = m.bridge.View()
	case screenBridgeForm:
		body = m.bridgeForm.View()
	case screenClientDashboard:
		body = m.clientDashboard.View()
	case screenAccountForm:
		body = m.accountForm.View()
	}

	if m.status != "" {
		style := statusBarStyle
		if m.statusErr {
			style = errorStyle
		}
		body += "\n" + style.Render(m.status)
	}

	if m.confirm.active {
		return borderStyle().Width(m.contentWidth()).Render(m.confirm.View())
	}
	if m.prompt.active {
		return borderStyle().Width(m.contentWidth()).Render(m.prompt.View())
	}

	return borderStyle().Width(m.contentWidth()).Render(body)
}

func (m Model) contentWidth() int {
	const min, max = 60, 100
	w := m.width - 4
	if w < min {
		return min
	}
	if w > max {
		return max
	}
	return w
}

func (m Model) viewProfileList() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render("softether-tui "+m.version))
	b.WriteString(tr("接続プロファイルを選択\n\n"))

	if len(m.profiles) == 0 {
		b.WriteString(dimStyle.Render(tr("プロファイルがありません。'a' で追加してください。")) + "\n")
	}

	for i, p := range m.profiles {
		marker := "  "
		style := statusBarStyle
		if i == m.cursor {
			marker = "> "
			style = selectedStyle
		}

		statusLabel := tr("未確認")
		if err, ok := m.testResults[p.Name]; ok {
			if err == nil {
				statusLabel = tr("● 接続確認済み")
			} else {
				statusLabel = tr("✕ ") + err.Error()
			}
		}

		line := fmt.Sprintf("%s%-16s %-24s %-8s %s", marker, p.Name, p.Address(), modeLabel(p.Mode), statusLabel)
		b.WriteString(style.Render(line) + "\n")
	}

	b.WriteString("\n" + dimStyle.Render(tr("↑/↓ j/k:選択  Enter:接続  a:追加  e:編集  d:削除  t:接続テスト  q:終了")))
	return b.String()
}

func modeLabel(mode config.Mode) string {
	switch mode {
	case config.ModeBridge:
		return "Bridge"
	case config.ModeClient:
		return "Client"
	default:
		return "Server"
	}
}
