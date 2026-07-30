// Package vpncmd wraps the SoftEther vpncmd binary as a non-interactive
// subprocess, per app_specs.md section 6.2.
package vpncmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"softether-tui/internal/config"
)

var ErrNotFound = errors.New("vpncmd: binary not found in PATH or executable directory")

// Locate finds the vpncmd binary. It checks:
// 1. Same directory as the current executable
// 2. Current working directory
// 3. System PATH
func Locate() (string, error) {
	if exePath, err := os.Executable(); err == nil {
		sameDirBinary := filepath.Join(filepath.Dir(exePath), "vpncmd")
		if info, err := os.Stat(sameDirBinary); err == nil && !info.IsDir() {
			return sameDirBinary, nil
		}
	}

	if info, err := os.Stat("./vpncmd"); err == nil && !info.IsDir() {
		if absPath, err := filepath.Abs("./vpncmd"); err == nil {
			return absPath, nil
		}
		return "./vpncmd", nil
	}

	path, err := exec.LookPath("vpncmd")
	if err != nil {
		return "", ErrNotFound
	}
	return path, nil
}

// Target identifies the server/hub a command should run against.
type Target struct {
	Host     string
	Port     int
	Mode     config.Mode
	Hub      string
	Password string
}

// Client runs vpncmd commands against a Target.
type Client struct {
	BinaryPath string
	Timeout    time.Duration
}

func NewClient(binaryPath string) *Client {
	return &Client{BinaryPath: binaryPath, Timeout: 15 * time.Second}
}

func buildArgs(t Target, command string, args []string) []string {
	out := []string{fmt.Sprintf("%s:%d", t.Host, t.Port)}
	switch t.Mode {
	case config.ModeBridge:
		out = append(out, "/BRIDGE")
	case config.ModeClient:
		out = append(out, "/CLIENT")
	default:
		out = append(out, "/SERVER")
	}
	// VPN Client accounts are not scoped to a Hub the way Server/Bridge
	// Hub Management Mode commands are.
	if t.Hub != "" && t.Mode != config.ModeClient {
		out = append(out, "/HUB:"+t.Hub)
	}
	out = append(out, "/CSV", "/CMD", command)
	out = append(out, args...)
	return out
}

// Run executes a single non-interactive vpncmd command and returns its
// stdout. The admin password, if any, is written to stdin rather than
// passed as a command-line argument so it does not appear in `ps` output.
func (c *Client) Run(ctx context.Context, t Target, command string, args ...string) (string, error) {
	return c.run(ctx, t, command, args, nil)
}

// RunWithInput is like Run but writes additional lines to stdin after the
// connection password, for commands that prompt for further input (e.g.
// HubCreate's initial admin password prompt).
func (c *Client) RunWithInput(ctx context.Context, t Target, command string, args []string, extraStdin []string) (string, error) {
	return c.run(ctx, t, command, args, extraStdin)
}

func (c *Client) run(ctx context.Context, t Target, command string, args []string, extraStdin []string) (string, error) {
	if c == nil || c.BinaryPath == "" {
		return "", ErrNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.BinaryPath, buildArgs(t, command, args)...)
	lines := append([]string{t.Password}, extraStdin...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("vpncmd %s failed: %w: %s", command, err, msg)
	}
	return stdout.String(), nil
}

// KeyValue holds the Item/Value pairs vpncmd's `/CSV` Get-style commands emit.
type KeyValue map[string]string

// ParseCSV parses a two-column "Item,Value" CSV payload into a KeyValue map.
func ParseCSV(output string) (KeyValue, error) {
	r := csv.NewReader(strings.NewReader(output))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	kv := make(KeyValue, len(records))
	for _, rec := range records {
		if len(rec) < 2 {
			continue
		}
		kv[rec[0]] = rec[1]
	}
	return kv, nil
}

// ServerInfo runs ServerInfoGet and returns its fields.
func (c *Client) ServerInfo(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "ServerInfoGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// ServerStatus runs ServerStatusGet and returns its fields.
func (c *Client) ServerStatus(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "ServerStatusGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// ServerPasswordSet sets the VPN Server administrator password.
func (c *Client) ServerPasswordSet(ctx context.Context, t Target, newPassword string) error {
	_, err := c.Run(ctx, t, "ServerPasswordSet", newPassword)
	return err
}

// Table holds the parsed result of a List-style vpncmd command, which emits
// one CSV header row followed by one row per item (Hub, User, Session, ...).
type Table struct {
	Headers []string
	Rows    []KeyValue
}

// NameOf returns the value of a row's first column. vpncmd's List-style CSV
// output conventionally puts the identifying name/ID in the first column
// (HubName, User name, ...), so this gives callers a name to key off of
// without depending on the exact (unverified) header text for each command.
func (t Table) NameOf(row KeyValue) string {
	if len(t.Headers) == 0 {
		return ""
	}
	return row[t.Headers[0]]
}

// ParseCSVTable parses a header-row-plus-data-rows CSV payload, as emitted
// by vpncmd's List-style commands (HubList, UserList, SessionList, ...).
func ParseCSVTable(output string) (Table, error) {
	r := csv.NewReader(strings.NewReader(output))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return Table{}, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) == 0 {
		return Table{}, nil
	}
	headers := records[0]
	rows := make([]KeyValue, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(KeyValue, len(headers))
		for i, h := range headers {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		rows = append(rows, row)
	}
	return Table{Headers: headers, Rows: rows}, nil
}

// hubScoped returns a copy of t with Hub cleared, for commands that operate
// in Server Admin Mode (hub name passed as a command argument) rather than
// Hub Management Mode (hub selected via the /HUB: connection option).
func hubScoped(t Target) Target {
	t.Hub = ""
	return t
}

// HubList runs HubList and returns the server's virtual hubs.
func (c *Client) HubList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, hubScoped(t), "HubList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

// HubGet fetches configuration/status for a specific hub using StatusGet command in Hub Management Mode.
func (c *Client) HubGet(ctx context.Context, t Target, hubName string) (KeyValue, error) {
	target := t
	target.Hub = hubName
	out, err := c.Run(ctx, target, "StatusGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// HubCreate creates a new virtual hub with the given initial admin password.
func (c *Client) HubCreate(ctx context.Context, t Target, hubName, hubPassword string) error {
	_, err := c.RunWithInput(ctx, hubScoped(t), "HubCreate", []string{hubName}, []string{hubPassword})
	return err
}

// HubDelete deletes a virtual hub.
func (c *Client) HubDelete(ctx context.Context, t Target, hubName string) error {
	_, err := c.Run(ctx, hubScoped(t), "HubDelete", hubName)
	return err
}

// HubSetOnline brings a hub online or offline. It scopes into the hub via
// the /HUB: connection option (equivalent to the interactive `Hub <name>`
// selection) and issues the Hub Management Mode Online/Offline command.
func (c *Client) HubSetOnline(ctx context.Context, t Target, hubName string, online bool) error {
	target := t
	target.Hub = hubName
	command := "Offline"
	if online {
		command = "Online"
	}
	_, err := c.Run(ctx, target, command)
	return err
}

// --- Hub Management Mode: users & groups (t.Hub must already name the
// target hub; callers scope Target via WithHub before calling these). ---

// WithHub returns a copy of t scoped to the given hub, for the Hub
// Management Mode commands below (UserList, GroupList, ...).
func (t Target) WithHub(hubName string) Target {
	t.Hub = hubName
	return t
}

// UserAuthType identifies the authentication method a user is created with.
// NTLM and certificate authentication are intentionally not offered here:
// their vpncmd parameter names are unconfirmed (see vpncmd_commands.md).
type UserAuthType int

const (
	UserAuthPassword UserAuthType = iota
	UserAuthAnonymous
	UserAuthRadius
)

// UserCreateOptions holds the optional fields UserCreate accepts.
type UserCreateOptions struct {
	Group    string
	RealName string
	Note     string
}

func (c *Client) UserList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "UserList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) UserGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "UserGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// UserCreate creates a user entry. The entry has no working authentication
// method until a matching UserXxxSet call configures one (UserPasswordSet,
// UserAnonymousSet, UserRadiusSet).
func (c *Client) UserCreate(ctx context.Context, t Target, name string, opts UserCreateOptions) error {
	args := []string{name}
	if opts.Group != "" {
		args = append(args, "/GROUP:"+opts.Group)
	}
	if opts.RealName != "" {
		args = append(args, "/REALNAME:"+opts.RealName)
	}
	if opts.Note != "" {
		args = append(args, "/NOTE:"+opts.Note)
	}
	_, err := c.Run(ctx, t, "UserCreate", args...)
	return err
}

func (c *Client) UserDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "UserDelete", name)
	return err
}

// UserPasswordSet switches the user to password authentication with the
// given password. The password is sent over stdin, not argv.
func (c *Client) UserPasswordSet(ctx context.Context, t Target, name, password string) error {
	_, err := c.RunWithInput(ctx, t, "UserPasswordSet", []string{name}, []string{password})
	return err
}

func (c *Client) UserAnonymousSet(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "UserAnonymousSet", name)
	return err
}

func (c *Client) UserRadiusSet(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "UserRadiusSet", name)
	return err
}

// UserSetGroup reassigns a user's group via UserSet's /GROUP: parameter.
// Pass an empty group to clear the assignment.
func (c *Client) UserSetGroup(ctx context.Context, t Target, name, group string) error {
	_, err := c.Run(ctx, t, "UserSet", name, "/GROUP:"+group)
	return err
}

// UserExpiresSet sets the account expiration date/time, using the
// "YYYY/MM/DD HH:MM:SS" format SoftEther uses elsewhere in vpncmd output.
func (c *Client) UserExpiresSet(ctx context.Context, t Target, name string, expires time.Time) error {
	_, err := c.Run(ctx, t, "UserExpiresSet", name, expires.Format("2006/01/02 15:04:05"))
	return err
}

// GroupCreateOptions holds the optional fields GroupCreate accepts.
type GroupCreateOptions struct {
	RealName string
	Note     string
}

func (c *Client) GroupList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "GroupList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) GroupGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "GroupGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

func (c *Client) GroupCreate(ctx context.Context, t Target, name string, opts GroupCreateOptions) error {
	args := []string{name}
	if opts.RealName != "" {
		args = append(args, "/REALNAME:"+opts.RealName)
	}
	if opts.Note != "" {
		args = append(args, "/NOTE:"+opts.Note)
	}
	_, err := c.Run(ctx, t, "GroupCreate", args...)
	return err
}

func (c *Client) GroupDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "GroupDelete", name)
	return err
}

// --- Hub Management Mode: sessions & log settings ---

func (c *Client) SessionList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "SessionList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) SessionGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "SessionGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

func (c *Client) SessionDisconnect(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "SessionDisconnect", name)
	return err
}

// LogGet returns the hub's current security/packet log settings. Changing
// those settings (LogPacketSaveType, LogSwitchType, ...) is not yet
// implemented: their exact vpncmd argument syntax is unconfirmed (see
// vpncmd_commands.md), so this client intentionally only offers the safe,
// well-understood Get-style read.
func (c *Client) LogGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "LogGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// --- Server Admin Mode: listeners (server-wide, not tied to a single hub) ---

func (c *Client) ListenerList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, hubScoped(t), "ListenerList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) ListenerCreate(ctx context.Context, t Target, port int) error {
	_, err := c.Run(ctx, hubScoped(t), "ListenerCreate", strconv.Itoa(port))
	return err
}

func (c *Client) ListenerDelete(ctx context.Context, t Target, port int) error {
	_, err := c.Run(ctx, hubScoped(t), "ListenerDelete", strconv.Itoa(port))
	return err
}

func (c *Client) ListenerEnable(ctx context.Context, t Target, port int) error {
	_, err := c.Run(ctx, hubScoped(t), "ListenerEnable", strconv.Itoa(port))
	return err
}

func (c *Client) ListenerDisable(ctx context.Context, t Target, port int) error {
	_, err := c.Run(ctx, hubScoped(t), "ListenerDisable", strconv.Itoa(port))
	return err
}

// --- Hub Management Mode: SecureNAT ---

func (c *Client) SecureNatEnable(ctx context.Context, t Target) error {
	_, err := c.Run(ctx, t, "SecureNatEnable")
	return err
}

func (c *Client) SecureNatDisable(ctx context.Context, t Target) error {
	_, err := c.Run(ctx, t, "SecureNatDisable")
	return err
}

func (c *Client) SecureNatStatusGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "SecureNatStatusGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// SecureNatHostGet returns the hub's virtual NAT host settings (IP range,
// DHCP, ...). Changing them via SecureNatHostSet is not yet implemented:
// its parameter set is large and unconfirmed (see vpncmd_commands.md).
func (c *Client) SecureNatHostGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "SecureNatHostGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

// --- Hub Management Mode: access list (packet filter) ---

func (c *Client) AccessList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "AccessList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

// AccessDelete, AccessEnable and AccessDisable take the rule identifier
// found in AccessList's first column (see Table.NameOf). AccessAdd (rule
// creation) is intentionally not implemented: its full parameter set
// (priority, discard/pass, source/destination address & port ranges,
// protocol) is unconfirmed and too risky to guess at (see
// vpncmd_commands.md).
func (c *Client) AccessDelete(ctx context.Context, t Target, id string) error {
	_, err := c.Run(ctx, t, "AccessDelete", id)
	return err
}

func (c *Client) AccessEnable(ctx context.Context, t Target, id string) error {
	_, err := c.Run(ctx, t, "AccessEnable", id)
	return err
}

func (c *Client) AccessDisable(ctx context.Context, t Target, id string) error {
	_, err := c.Run(ctx, t, "AccessDisable", id)
	return err
}

// --- Hub Management Mode: cascade connections (site-to-site, Bridge use) ---

func (c *Client) CascadeList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "CascadeList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) CascadeStatusGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "CascadeStatusGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

func (c *Client) CascadeDetailGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "CascadeDetailGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

func (c *Client) CascadeDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "CascadeDelete", name)
	return err
}

// CascadeSetOnline brings a named cascade connection online or offline.
// CascadeCreate (establishing a new cascade to a remote server) is
// intentionally not implemented: it requires a remote host/port/hub and an
// authentication method, and the exact vpncmd flag names for that are
// unconfirmed (see vpncmd_commands.md).
func (c *Client) CascadeSetOnline(ctx context.Context, t Target, name string, online bool) error {
	command := "CascadeOffline"
	if online {
		command = "CascadeOnline"
	}
	_, err := c.Run(ctx, t, command, name)
	return err
}

// --- Server Admin Mode: local bridges (server-wide, tied to a physical NIC/tap) ---

func (c *Client) BridgeDeviceList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, hubScoped(t), "BridgeDeviceList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) BridgeList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, hubScoped(t), "BridgeList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) BridgeCreate(ctx context.Context, t Target, hubName, deviceName string, tap bool) error {
	tapFlag := "no"
	if tap {
		tapFlag = "yes"
	}
	_, err := c.Run(ctx, hubScoped(t), "BridgeCreate", hubName, "/DEVICE:"+deviceName, "/TAP:"+tapFlag)
	return err
}

func (c *Client) BridgeDelete(ctx context.Context, t Target, hubName string) error {
	_, err := c.Run(ctx, hubScoped(t), "BridgeDelete", hubName)
	return err
}

// --- VPN Client Mode: connection accounts (Target.Mode must be ModeClient) ---

// AccountAuthType identifies the authentication method a VPN Client account
// connects with. Certificate authentication (AccountCertSet) is
// intentionally not offered here: its vpncmd parameter name/shape is
// unconfirmed (see vpncmd_commands.md), same reasoning as UserAuthType.
type AccountAuthType int

const (
	AccountAuthPassword AccountAuthType = iota
	AccountAuthAnonymous
)

// AccountCreateOptions holds the fields AccountCreate needs to point a new
// connection account at a remote server/hub.
type AccountCreateOptions struct {
	ServerHost string
	ServerPort int
	Hub        string
}

func (c *Client) AccountList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "AccountList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

func (c *Client) AccountStatusGet(ctx context.Context, t Target, name string) (KeyValue, error) {
	out, err := c.Run(ctx, t, "AccountStatusGet", name)
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

func (c *Client) AccountCreate(ctx context.Context, t Target, name string, opts AccountCreateOptions) error {
	args := []string{
		name,
		fmt.Sprintf("/SERVER:%s:%d", opts.ServerHost, opts.ServerPort),
		"/HUB:" + opts.Hub,
	}
	_, err := c.Run(ctx, t, "AccountCreate", args...)
	return err
}

func (c *Client) AccountDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "AccountDelete", name)
	return err
}

func (c *Client) AccountUsernameSet(ctx context.Context, t Target, name, username string) error {
	_, err := c.Run(ctx, t, "AccountUsernameSet", name, username)
	return err
}

// AccountPasswordSet switches the account to password authentication with
// the given password, sent over stdin rather than argv.
func (c *Client) AccountPasswordSet(ctx context.Context, t Target, name, password string) error {
	_, err := c.RunWithInput(ctx, t, "AccountPasswordSet", []string{name}, []string{password})
	return err
}

func (c *Client) AccountAnonymousSet(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "AccountAnonymousSet", name)
	return err
}

func (c *Client) AccountConnect(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "AccountConnect", name)
	return err
}

func (c *Client) AccountDisconnect(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "AccountDisconnect", name)
	return err
}
