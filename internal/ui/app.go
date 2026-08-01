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
	screenUserDetail
	screenGroupForm
	screenGroupDetail
	screenSecureNATDetail
	screenRadiusForm
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
	radiusForm  *radiusForm
	bridgeForm  *bridgeForm
	accountForm *accountForm

	dashboard       dashboardState
	hubDetail       hubDetailState
	userDetail      userDetailState
	groupDetail     groupDetailState
	secureNatDetail secureNatDetailState
	listener        listenerState
	bridge          bridgeState
	clientDashboard clientDashboardState

	status    string
	statusErr bool

	sessionPasswords        map[string]string
	initialPasswordPrompted map[string]bool

	quitting bool
	width    int
	height   int
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
		radiusForm:              newRadiusForm(),
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		type resStruct struct {
			info   vpncmd.KeyValue
			status vpncmd.KeyValue
			hubs   vpncmd.Table
			err    error
		}

		infoChan := make(chan resStruct, 1)
		statusChan := make(chan resStruct, 1)
		hubsChan := make(chan resStruct, 1)

		go func() {
			info, err := client.ServerInfo(ctx, target)
			infoChan <- resStruct{info: info, err: err}
		}()

		go func() {
			status, err := client.ServerStatus(ctx, target)
			statusChan <- resStruct{status: status, err: err}
		}()

		go func() {
			hubs, err := client.HubList(ctx, target)
			hubsChan <- resStruct{hubs: hubs, err: err}
		}()

		r1 := <-infoChan
		if r1.err != nil {
			return serverInfoMsg{profileName: name, err: r1.err}
		}

		r2 := <-statusChan
		if r2.err != nil {
			return serverInfoMsg{profileName: name, err: r2.err}
		}

		r3 := <-hubsChan
		if r3.err != nil {
			return serverInfoMsg{profileName: name, err: r3.err}
		}

		return serverInfoMsg{profileName: name, info: r1.info, status: r2.status, hubs: r3.hubs}
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

type hubPasswordResultMsg struct {
	hubName string
	err     error
}

func (m Model) setHubPassword(p config.Profile, hubName, password string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.SetHubPassword(ctx, target, hubName, password)
		return hubPasswordResultMsg{hubName: hubName, err: err}
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
	if p.Password != "" {
		return p.Password
	}
	return ""
}

// --- update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
		if msg.err != nil && (strings.Contains(msg.err.Error(), "Access has been denied") || strings.Contains(msg.err.Error(), "exit status 1")) {
			// Clear invalid password from session & saved profile so prompt is re-shown
			delete(m.sessionPasswords, p.Name)
			for i, prof := range m.profiles {
				if prof.Name == p.Name && prof.Password != "" {
					m.profiles[i].Password = ""
					_ = m.store.Save(m.profiles)
					break
				}
			}
			m.dashboard.loading = false
			m.dashboard.err = msg.err
			m.prompt.Show(promptConnectPassword, p.Name, fmt.Sprintf(tr("管理者パスワードを入力してください (%s)"), p.Name), tr("パスワード"), true)
			return m, nil
		}

		// On successful connection, persist the working password in profile if entered via prompt
		if msg.err == nil {
			if pw, ok := m.sessionPasswords[p.Name]; ok && pw != "" {
				for i, prof := range m.profiles {
					if prof.Name == p.Name && prof.Password != pw {
						m.profiles[i].Password = pw
						m.dashboard.profile.Password = pw
						_ = m.store.Save(m.profiles)
						break
					}
				}
			}
		}

		currentPw := m.passwordFromEnvOrSession(p)

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

	case hubPasswordResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf(tr("Hub %q のパスワード設定に失敗しました: %s"), msg.hubName, msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q のパスワードを設定しました"), msg.hubName)
		m.statusErr = false
		return m, nil

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
			m.hubDetail.secureNatDhcp = msg.dhcp
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

	case userDetailLoadedMsg:
		m.userDetail.loading = false
		m.userDetail.err = msg.err
		m.userDetail.info = msg.info
		return m, nil

	case groupDetailLoadedMsg:
		m.groupDetail.loading = false
		m.groupDetail.err = msg.err
		m.groupDetail.info = msg.info
		return m, nil

	case secureNatDetailLoadedMsg:
		m.secureNatDetail.loading = false
		m.secureNatDetail.err = msg.err
		m.secureNatDetail.status = msg.status
		m.secureNatDetail.host = msg.host
		m.secureNatDetail.dhcp = msg.dhcp
		return m, nil

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
	case screenUserDetail:
		return m.handleUserDetailKey(msg)
	case screenGroupForm:
		return m.handleGroupFormKey(msg)
	case screenRadiusForm:
		return m.handleRadiusFormKey(msg)
	case screenGroupDetail:
		return m.handleGroupDetailKey(msg)
	case screenSecureNATDetail:
		return m.handleSecureNATDetailKey(msg)
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

	case promptAccountUsername:
		m.status = fmt.Sprintf(tr("%s のユーザー名を変更しています..."), target)
		m.statusErr = false
		return m, m.setAccountUsername(m.clientDashboard.profile, target, value)

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

	case promptAccessAddMemo:
		memo := strings.TrimSpace(value)
		if memo == "" {
			memo = "Custom Rule"
		}
		m.status = fmt.Sprintf(tr("アクセスリストルール %q を追加しています..."), memo)
		m.statusErr = false
		opts := vpncmd.AccessAddOptions{
			Pass:     true,
			Memo:     memo,
			Priority: 100,
		}
		return m, m.addAccessRule(profile, hub, opts)

	case promptCascadeCreateName:
		name := strings.TrimSpace(value)
		if name != "" {
			m.status = fmt.Sprintf(tr("カスケード接続 %q を作成しています..."), name)
			m.statusErr = false
			opts := vpncmd.CascadeCreateOptions{
				Name:       name,
				ServerHost: profile.Host,
				ServerPort: profile.Port,
				Hub:        hub,
				User:       "cascade",
			}
			return m, m.createCascade(profile, hub, opts)
		}

	case promptHubPassword:
		m.status = fmt.Sprintf(tr("Hub %q のパスワードを設定しています..."), target)
		m.statusErr = false
		return m, m.setHubPassword(m.dashboard.profile, target, value)
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

	case confirmToggleSecureNAT:
		d := &m.hubDetail
		snEnabled := false
		for _, k := range []string{"SecureNAT Functionality State", "SecureNAT Status", "Status", "SecureNAT"} {
			if v, ok := d.secureNatStatus[k]; ok {
				vLower := strings.ToLower(v)
				if strings.Contains(vLower, "enable") || strings.Contains(vLower, "active") || strings.Contains(vLower, "running") || strings.Contains(vLower, "yes") {
					snEnabled = true
					break
				}
			}
		}
		actionLabel := tr("有効化")
		if snEnabled {
			actionLabel = tr("無効化")
		}
		m.status = fmt.Sprintf(tr("Hub %q の SecureNAT を%sしています..."), target, actionLabel)
		m.statusErr = false
		return m, m.setSecureNatEnabled(d.profile, target, !snEnabled)

	case confirmToggleDHCP:
		d := &m.hubDetail
		dhcpEnabled := false
		for _, k := range []string{"Use Virtual DHCP Server", "Virtual DHCP Server", "Use DHCP", "DHCP Server", "Status"} {
			if v, ok := d.secureNatDhcp[k]; ok {
				vLower := strings.ToLower(v)
				if strings.Contains(vLower, "yes") || strings.Contains(vLower, "enable") || strings.Contains(vLower, "active") || strings.Contains(vLower, "true") {
					dhcpEnabled = true
					break
				}
			}
		}
		actionLabel := tr("有効化")
		if dhcpEnabled {
			actionLabel = tr("無効化")
		}
		m.status = fmt.Sprintf(tr("Hub %q の Virtual DHCP を%sしています..."), target, actionLabel)
		m.statusErr = false
		return m, m.setDhcpEnabled(d.profile, target, !dhcpEnabled)

	case confirmEnableListener:
		m.status = fmt.Sprintf(tr("リスナー %q を有効化しています..."), target)
		m.statusErr = false
		return m, m.setListenerEnabled(m.listener.profile, target, true)

	case confirmDisableListener:
		m.status = fmt.Sprintf(tr("リスナー %q を無効化しています..."), target)
		m.statusErr = false
		return m, m.setListenerEnabled(m.listener.profile, target, false)
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

	case "n":
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

	case "n":
		m.hubForm.Reset()
		m.screen = screenHubForm
		m.status = ""

	case "d":
		if name, ok := m.currentHubName(); ok {
			m.confirm.Show(confirmDeleteHub, name, fmt.Sprintf(tr("Hub %q を削除しますか?"), name))
		}

	case "p":
		if name, ok := m.currentHubName(); ok {
			m.prompt.Show(promptHubPassword, name, fmt.Sprintf(tr("Hub %q の新しいパスワードを入力してください:"), name), tr("パスワード"), true)
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

	case "tab", "right":
		if m.hubDetail.tab == hubTabSecureNAT && m.hubDetail.secureNatEditing {
			return m.handleHubSecureNATKey(msg)
		}
		m.status = ""
		m.statusErr = false
		m.hubDetail.tab = (m.hubDetail.tab + 1) % hubTabCount
		return m.loadHubTabIfNeeded()

	case "shift+tab", "left":
		if m.hubDetail.tab == hubTabSecureNAT && m.hubDetail.secureNatEditing {
			return m.handleHubSecureNATKey(msg)
		}
		m.status = ""
		m.statusErr = false
		m.hubDetail.tab = (m.hubDetail.tab - 1 + hubTabCount) % hubTabCount
		return m.loadHubTabIfNeeded()

	case "r":
		if m.hubDetail.tab == hubTabSecureNAT && m.hubDetail.secureNatEditing {
			return m.handleHubSecureNATKey(msg)
		}
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
	case hubTabLog:
		return m.handleHubLogKey(msg)
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
	case "r", "R":
		m.radiusForm.Reset()
		m.screen = screenRadiusForm
		m.status = ""
		return m, nil

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

	case "enter":
		if name, ok := m.hubDetail.currentUserName(); ok {
			m.userDetail = userDetailState{profile: m.hubDetail.profile, hubName: m.hubDetail.hubName, userName: name, loading: true}
			m.screen = screenUserDetail
			m.status = ""
			return m, m.fetchUserDetail(m.hubDetail.profile, m.hubDetail.hubName, name)
		}

	case "n":
		m.userForm.Reset()
		var groupNames []string
		for _, row := range m.hubDetail.groups.Rows {
			if gName := m.hubDetail.groups.NameOf(row); gName != "" {
				groupNames = append(groupNames, gName)
			}
		}
		m.userForm.SetGroups(groupNames)
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

	case "enter":
		if name, ok := m.hubDetail.currentGroupName(); ok {
			m.groupDetail = groupDetailState{profile: m.hubDetail.profile, hubName: m.hubDetail.hubName, groupName: name, loading: true}
			m.screen = screenGroupDetail
			m.status = ""
			return m, m.fetchGroupDetail(m.hubDetail.profile, m.hubDetail.hubName, name)
		}

	case "n":
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

func (m Model) handleHubLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.hubDetail
	total := len(logSettingKeys)

	switch msg.String() {
	case "up", "k":
		if d.logCursor > 0 {
			d.logCursor--
		}

	case "down", "j":
		if d.logCursor < total-1 {
			d.logCursor++
		}

	case "enter", " ", "space":
		if d.logCursor >= 0 && d.logCursor < total {
			return m.toggleLogSetting(d.logCursor)
		}
	}
	return m, nil
}

func (m Model) toggleLogSetting(cursor int) (tea.Model, tea.Cmd) {
	p := m.hubDetail.profile
	hub := m.hubDetail.hubName
	item := logSettingKeys[cursor]
	curr := m.hubDetail.getLogKV(item.key)

	var cmd tea.Cmd
	switch item.key {
	case "Save Security Log":
		enable := !strings.EqualFold(curr, "Enable")
		m.status = fmt.Sprintf(tr("Hub %q のセキュリティログを%sに設定中..."), hub, map[bool]string{true: tr("有効"), false: tr("無効")}[enable])
		m.statusErr = false
		cmd = m.setLogEnableDisable(p, hub, "security", enable)

	case "Save Packet Log":
		enable := !strings.EqualFold(curr, "Enable")
		m.status = fmt.Sprintf(tr("Hub %q のパケットログを%sに設定中..."), hub, map[bool]string{true: tr("有効"), false: tr("無効")}[enable])
		m.statusErr = false
		cmd = m.setLogEnableDisable(p, hub, "packet", enable)

	case "Security Switch Cycle":
		nextCycle := cycleNextLogSwitch(curr)
		m.status = fmt.Sprintf(tr("Hub %q のセキュリティログ切り替え周期を %s に設定中..."), hub, nextCycle)
		m.statusErr = false
		cmd = m.setLogSwitch(p, hub, "security", nextCycle)

	case "Packet Switch Cycle":
		nextCycle := cycleNextLogSwitch(curr)
		m.status = fmt.Sprintf(tr("Hub %q のパケットログ切り替え周期を %s に設定中..."), hub, nextCycle)
		m.statusErr = false
		cmd = m.setLogSwitch(p, hub, "packet", nextCycle)

	default:
		// Packet save type settings: tcpconn, tcpdata, dhcp, udp, icmp, ip, arp, ether
		pTypeMap := map[string]string{
			"TCP Connection Log": "tcpconn",
			"TCP Packet Log":     "tcpdata",
			"DHCP Log":           "dhcp",
			"UDP Log":            "udp",
			"ICMP Log":           "icmp",
			"IP Log":             "ip",
			"ARP Log":            "arp",
			"Ethernet Log":       "ether",
		}
		pType, ok := pTypeMap[item.key]
		if ok {
			nextSave := cycleNextPacketSave(curr)
			m.status = fmt.Sprintf(tr("Hub %q の %s ログ保存形式を %s に設定中..."), hub, pType, nextSave)
			m.statusErr = false
			cmd = m.setLogPacketSave(p, hub, pType, nextSave)
		}
	}

	return m, tea.Batch(cmd, m.fetchLog(p, hub))
}

func cycleNextLogSwitch(curr string) string {
	// cycles: day -> month -> sec -> min -> hour -> none -> day
	switch strings.ToLower(curr) {
	case "switch in every day", "day":
		return "month"
	case "switch in every month", "month":
		return "none"
	case "no switch", "none":
		return "sec"
	case "switch in every second", "sec":
		return "min"
	case "switch in every minute", "min":
		return "hour"
	default:
		return "day"
	}
}

func cycleNextPacketSave(curr string) string {
	// cycles: Do not Save / none -> Header Only / header -> Save All / full -> none
	switch strings.ToLower(curr) {
	case "do not save", "none":
		return "header"
	case "header only", "header":
		return "full"
	default:
		return "none"
	}
}

func (m Model) setLogEnableDisable(p config.Profile, hub, logType string, enable bool) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		if enable {
			err = client.LogEnable(ctx, target, logType)
		} else {
			err = client.LogDisable(ctx, target, logType)
		}
		return secureNatActionResultMsg{action: tr("ログ有効/無効切り替え"), err: err}
	}
}

func (m Model) setLogSwitch(p config.Profile, hub, logType, switchCycle string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.LogSwitchSet(ctx, target, logType, switchCycle)
		return secureNatActionResultMsg{action: tr("ログ切り替え周期変更"), err: err}
	}
}

func (m Model) setLogPacketSave(p config.Profile, hub, packetType, saveType string) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.LogPacketSaveType(ctx, target, packetType, saveType)
		return secureNatActionResultMsg{action: tr("パケットログ保存形式変更"), err: err}
	}
}

func (m Model) handleHubSecureNATKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &m.hubDetail

	if d.secureNatEditing {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(d.filterInput.Value())
			if d.secureNatEditedValues == nil {
				d.secureNatEditedValues = make(map[editableSecureNATField]string)
			}
			d.secureNatEditedValues[d.secureNatEditingField] = val
			d.secureNatDirty = true
			d.secureNatEditing = false
			return m, nil

		case "esc":
			d.secureNatEditing = false
			return m, nil
		}
		var cmd tea.Cmd
		d.filterInput, cmd = d.filterInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if d.secureNatCursor > 0 {
			d.secureNatCursor--
		}

	case "down", "j":
		if d.secureNatCursor < editableSecureNATFieldCount-1 {
			d.secureNatCursor++
		}

	case "enter":
		if d.secureNatCursor == fieldSecureNAT {
			snEnabled := false
			for _, k := range []string{"SecureNAT Functionality State", "SecureNAT Status", "Status", "SecureNAT"} {
				if v, ok := d.secureNatStatus[k]; ok {
					vLower := strings.ToLower(v)
					if strings.Contains(vLower, "enable") || strings.Contains(vLower, "active") || strings.Contains(vLower, "running") || strings.Contains(vLower, "yes") {
						snEnabled = true
						break
					}
				}
			}
			actionLabel := tr("有効化")
			if snEnabled {
				actionLabel = tr("無効化")
			}
			m.confirm.Show(confirmToggleSecureNAT, m.hubDetail.hubName, fmt.Sprintf(tr("Hub %q の SecureNAT を%sしますか?"), m.hubDetail.hubName, actionLabel))
			return m, nil
		}
		if d.secureNatCursor == fieldDHCP {
			dhcpEnabled := false
			for _, k := range []string{"Use Virtual DHCP Server", "Virtual DHCP Server", "Use DHCP", "DHCP Server", "Status"} {
				if v, ok := d.secureNatDhcp[k]; ok {
					vLower := strings.ToLower(v)
					if strings.Contains(vLower, "yes") || strings.Contains(vLower, "enable") || strings.Contains(vLower, "active") || strings.Contains(vLower, "true") {
						dhcpEnabled = true
						break
					}
				}
			}
			actionLabel := tr("有効化")
			if dhcpEnabled {
				actionLabel = tr("無効化")
			}
			m.confirm.Show(confirmToggleDHCP, m.hubDetail.hubName, fmt.Sprintf(tr("Hub %q の Virtual DHCP を%sしますか?"), m.hubDetail.hubName, actionLabel))
			return m, nil
		}

		d.secureNatEditing = true
		d.secureNatEditingField = d.secureNatCursor
		ti := textinput.New()
		if prev, ok := d.secureNatEditedValues[d.secureNatCursor]; ok {
			ti.SetValue(prev)
		} else {
			val := d.getNatFieldValue(d.secureNatCursor)
			if val == "(None)" {
				val = ""
			}
			ti.SetValue(val)
		}
		ti.Focus()
		d.filterInput = ti
		return m, nil

	case "s", "S":
		if d.secureNatDirty {
			return m.saveHubSecureNATChanges()
		}

	case "c", "C":
		if d.secureNatDirty {
			d.secureNatEditedValues = make(map[editableSecureNATField]string)
			d.secureNatDirty = false
			m.status = tr("変更を破棄しました")
			m.statusErr = false
			return m, nil
		}
	}
	return m, nil
}

func (d hubDetailState) getNatFieldValue(field editableSecureNATField) string {
	switch field {
	case fieldNatIP:
		return d.getNatHostKV("IP Address", "IP")
	case fieldNatMask:
		return d.getNatHostKV("Subnet Mask", "Mask")
	case fieldNatMAC:
		return d.getNatHostKV("MAC Address", "MAC")
	case fieldNatMTU:
		return d.getNatHostKV("MTU", "Mtu")
	case fieldDhcpRange:
		startIp := d.getNatDhcpKV("Start Distribution Address Band", "Start")
		endIp := d.getNatDhcpKV("End Distribution Address Band", "End")
		if startIp != "" && endIp != "" {
			return startIp + " - " + endIp
		}
		return startIp
	case fieldDhcpLease:
		return d.getNatDhcpKV("Lease Limit (Seconds)", "Lease")
	case fieldDhcpGW:
		return d.getNatDhcpKV("Default Gateway Address", "Gateway", "GW")
	case fieldDhcpDNS1:
		return d.getNatDhcpKV("DNS Server Address 1", "DNS")
	case fieldDhcpDNS2:
		return d.getNatDhcpKV("DNS Server Address 2", "DNS2")
	case fieldDhcpDomain:
		return d.getNatDhcpKV("Domain Name", "Domain")
	}
	return ""
}

func (m Model) saveHubSecureNATChanges() (tea.Model, tea.Cmd) {
	d := &m.hubDetail
	p := d.profile
	hub := d.hubName
	var cmds []tea.Cmd

	ip := d.secureNatEditedValues[fieldNatIP]
	mask := d.secureNatEditedValues[fieldNatMask]
	mac := d.secureNatEditedValues[fieldNatMAC]

	if ip != "" || mask != "" || mac != "" {
		hostOpts := vpncmd.SecureNatHostOptions{
			IP:   ip,
			Mask: mask,
			MAC:  mac,
		}
		cmds = append(cmds, m.setSecureNatHostOpts(p, hub, hostOpts))
	}

	rangeVal := d.secureNatEditedValues[fieldDhcpRange]
	lease := d.secureNatEditedValues[fieldDhcpLease]
	gw := d.secureNatEditedValues[fieldDhcpGW]
	dns1 := d.secureNatEditedValues[fieldDhcpDNS1]
	dns2 := d.secureNatEditedValues[fieldDhcpDNS2]
	domain := d.secureNatEditedValues[fieldDhcpDomain]

	if rangeVal != "" || lease != "" || gw != "" || dns1 != "" || dns2 != "" || domain != "" {
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
			GW:     gw,
			DNS:    dns1,
			DNS2:   dns2,
			Domain: domain,
		}
		cmds = append(cmds, m.setDhcpOpts(p, hub, dhcpOpts))
	}

	d.secureNatDirty = false
	d.secureNatEditedValues = make(map[editableSecureNATField]string)
	m.status = fmt.Sprintf(tr("Hub %q の SecureNAT 設定を保存しています..."), hub)
	m.statusErr = false

	cmds = append(cmds, m.fetchSecureNAT(p, hub))
	return m, tea.Batch(cmds...)
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

func (m Model) handleRadiusFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHubDetail
		return m, nil

	case "enter":
		serverPort, opts, err := m.radiusForm.Build()
		if err != nil {
			m.status = err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf(tr("Hub %q の RADIUS サーバーを設定しています..."), m.hubDetail.hubName)
		m.statusErr = false
		m.screen = screenHubDetail
		return m, m.setRadiusServer(m.hubDetail.profile, m.hubDetail.hubName, serverPort, opts)
	}

	cmd := m.radiusForm.Update(msg)
	return m, cmd
}

func (m Model) setRadiusServer(p config.Profile, hub, serverPort string, opts vpncmd.RadiusServerSetOptions) tea.Cmd {
	client := m.client
	target := m.targetFromProfile(p).WithHub(hub)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.RadiusServerSet(ctx, target, serverPort, opts)
		return secureNatActionResultMsg{action: tr("RADIUSサーバー設定"), err: err}
	}
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
	case screenUserDetail:
		body = m.userDetail.View()
	case screenGroupForm:
		body = m.groupForm.View()
	case screenRadiusForm:
		body = m.radiusForm.View()
	case screenGroupDetail:
		body = m.groupDetail.View()
	case screenSecureNATDetail:
		body = m.secureNatDetail.View()
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
		statusLine := style.Render(m.status)

		// Insert status message right below header line (first occurrence of ─── or newline)
		lines := strings.Split(body, "\n")
		inserted := false
		var newLines []string
		for i, line := range lines {
			newLines = append(newLines, line)
			if !inserted && (strings.Contains(line, "─") || (i == 1 && strings.TrimSpace(line) == "")) {
				newLines = append(newLines, statusLine, "")
				inserted = true
			}
		}
		if !inserted {
			newLines = append([]string{statusLine, ""}, lines...)
		}
		body = strings.Join(newLines, "\n")
	}

	style := borderStyle().Width(m.contentWidth())
	if m.height > 2 {
		targetContentHeight := m.height - 4 // minus top/bottom borders (2 lines) and padding
		lines := strings.Split(body, "\n")
		if len(lines) < targetContentHeight {
			// Pad blank lines between main content and footer help line(s)
			// Assuming footer is the last 1 or 2 lines after a newline
			var contentLines, footerLines []string
			// Find footer split point: renderHelp lines at end
			lastEmptyIndex := -1
			for i := len(lines) - 1; i >= 0; i-- {
				if lines[i] == "" {
					lastEmptyIndex = i
					break
				}
			}
			if lastEmptyIndex > 0 && lastEmptyIndex >= len(lines)-3 {
				contentLines = lines[:lastEmptyIndex]
				footerLines = lines[lastEmptyIndex:]
				padding := make([]string, targetContentHeight-len(contentLines)-len(footerLines))
				body = strings.Join(append(append(contentLines, padding...), footerLines...), "\n")
			}
		}
		style = style.Height(m.height - 2)
	}

	if m.confirm.active {
		return style.Render(m.confirm.View())
	}
	if m.prompt.active {
		return style.Render(m.prompt.View())
	}

	return style.Render(body)
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
		b.WriteString(dimStyle.Render(tr("プロファイルがありません。'n' で追加してください。")) + "\n")
	} else {
		// Table Header
		header := fmt.Sprintf("  %-16s %-24s %-8s %s", tr("接続名 (Name)"), tr("接続先 (Host:Port)"), tr("モード"), tr("状態 (Status)"))
		b.WriteString(headerStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  "+strings.Repeat("─", 68)) + "\n")

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
					statusLabel = tr("[OK] 接続確認済み")
				} else {
					statusLabel = tr("[ERR] ") + err.Error()
				}
			}

			line := fmt.Sprintf("%s%-16s %-24s %-8s %s", marker, p.Name, p.Address(), modeLabel(p.Mode), statusLabel)
			b.WriteString(style.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + renderHelp(
		"↑/↓ j/k", tr("選択"),
		"Enter", tr("接続"),
		"n", tr("新規追加"),
		"e", tr("編集"),
		"d", tr("削除"),
		"t", tr("接続テスト"),
		"q", tr("終了"),
	))
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
