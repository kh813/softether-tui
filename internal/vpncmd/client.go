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
	return &Client{BinaryPath: binaryPath, Timeout: 5 * time.Second}
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
		outStr := stdout.String()
		errStr := stderr.String()
		msg := strings.TrimSpace(errStr)
		if msg == "" {
			msg = strings.TrimSpace(outStr)
		}
		if strings.Contains(msg, "Access has been denied") {
			return "", fmt.Errorf("Access has been denied")
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

	// Filter out non-CSV lines (e.g. "Password:" prompt line) before locating header row
	var cleanRecords [][]string
	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}
		// Ignore single field records that look like prompts (e.g. "Password:", "Password: ****")
		if len(rec) == 1 && (strings.HasPrefix(strings.TrimSpace(rec[0]), "Password:") || strings.TrimSpace(rec[0]) == "") {
			continue
		}
		cleanRecords = append(cleanRecords, rec)
	}

	if len(cleanRecords) == 0 {
		return Table{}, nil
	}

	headers := cleanRecords[0]
	rows := make([]KeyValue, 0, len(cleanRecords)-1)
	for _, rec := range cleanRecords[1:] {
		row := make(KeyValue, len(headers))
		hasValue := false
		for i, h := range headers {
			if i < len(rec) {
				val := strings.TrimSpace(rec[i])
				if val != "" {
					hasValue = true
				}
				row[h] = rec[i]
			}
		}
		if hasValue {
			rows = append(rows, row)
		}
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
	args := []string{hubName}
	if hubPassword != "" {
		args = append(args, "/PASSWORD:"+hubPassword)
	}
	_, err := c.Run(ctx, hubScoped(t), "HubCreate", args...)
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
	group := opts.Group
	if group == "" {
		group = "none"
	}
	realName := opts.RealName
	if realName == "" {
		realName = "none"
	}
	note := opts.Note
	if note == "" {
		note = "none"
	}
	args := []string{name, "/GROUP:" + group, "/REALNAME:" + realName, "/NOTE:" + note}
	_, err := c.Run(ctx, t, "UserCreate", args...)
	return err
}

func (c *Client) UserDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "UserDelete", name)
	return err
}

// UserPasswordSet switches the user to password authentication with the
// given password.
func (c *Client) UserPasswordSet(ctx context.Context, t Target, name, password string) error {
	args := []string{name}
	if password != "" {
		args = append(args, "/PASSWORD:"+password)
	}
	_, err := c.Run(ctx, t, "UserPasswordSet", args...)
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

type UserSetOptions struct {
	Group    string
	RealName string
	Note     string
}

// UserSetGroup reassigns a user's group via UserSet's /GROUP: parameter.
// Pass an empty group to clear the assignment.
func (c *Client) UserSetGroup(ctx context.Context, t Target, name, group string) error {
	if group == "" {
		group = "none"
	}
	_, err := c.Run(ctx, t, "UserSet", name, "/GROUP:"+group)
	return err
}

func (c *Client) UserSet(ctx context.Context, t Target, name string, opts UserSetOptions) error {
	group := opts.Group
	if group == "" {
		group = "none"
	}
	realName := opts.RealName
	if realName == "" {
		realName = "none"
	}
	note := opts.Note
	if note == "" {
		note = "none"
	}
	args := []string{name, "/GROUP:" + group, "/REALNAME:" + realName, "/NOTE:" + note}
	_, err := c.Run(ctx, t, "UserSet", args...)
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
	realName := opts.RealName
	if realName == "" {
		realName = "none"
	}
	note := opts.Note
	if note == "" {
		note = "none"
	}
	args := []string{name, "/REALNAME:" + realName, "/NOTE:" + note}
	_, err := c.Run(ctx, t, "GroupCreate", args...)
	return err
}

type GroupSetOptions struct {
	RealName string
	Note     string
}

func (c *Client) GroupSet(ctx context.Context, t Target, name string, opts GroupSetOptions) error {
	realName := opts.RealName
	if realName == "" {
		realName = "none"
	}
	note := opts.Note
	if note == "" {
		note = "none"
	}
	args := []string{name, "/REALNAME:" + realName, "/NOTE:" + note}
	_, err := c.Run(ctx, t, "GroupSet", args...)
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

func (c *Client) SecureNatHostGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "SecureNatHostGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

type SecureNatHostOptions struct {
	MAC  string
	IP   string
	Mask string
}

func (c *Client) SecureNatHostSet(ctx context.Context, t Target, opts SecureNatHostOptions) error {
	args := []string{}
	if opts.MAC != "" {
		args = append(args, "/MAC:"+opts.MAC)
	}
	if opts.IP != "" {
		args = append(args, "/IP:"+opts.IP)
	}
	if opts.Mask != "" {
		args = append(args, "/MASK:"+opts.Mask)
	}
	_, err := c.Run(ctx, t, "SecureNatHostSet", args...)
	return err
}

func (c *Client) DhcpGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "DhcpGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

type DhcpSetOptions struct {
	Start  string
	End    string
	Mask   string
	Expire string
	GW     string
	DNS    string
	DNS2   string
	Domain string
}

func (c *Client) DhcpSet(ctx context.Context, t Target, opts DhcpSetOptions) error {
	args := []string{}
	if opts.Start != "" {
		args = append(args, "/START:"+opts.Start)
	}
	if opts.End != "" {
		args = append(args, "/END:"+opts.End)
	}
	if opts.Mask != "" {
		args = append(args, "/MASK:"+opts.Mask)
	}
	if opts.Expire != "" {
		args = append(args, "/EXPIRE:"+opts.Expire)
	}
	if opts.GW != "" {
		args = append(args, "/GW:"+opts.GW)
	}
	if opts.DNS != "" {
		args = append(args, "/DNS:"+opts.DNS)
	}
	if opts.DNS2 != "" {
		args = append(args, "/DNS2:"+opts.DNS2)
	}
	if opts.Domain != "" {
		args = append(args, "/DOMAIN:"+opts.Domain)
	}
	_, err := c.Run(ctx, t, "DhcpSet", args...)
	return err
}

func (c *Client) RadiusServerGet(ctx context.Context, t Target) (KeyValue, error) {
	out, err := c.Run(ctx, t, "RadiusServerGet")
	if err != nil {
		return nil, err
	}
	return ParseCSV(out)
}

type RadiusServerSetOptions struct {
	Secret        string
	RetryInterval string
}

func (c *Client) RadiusServerSet(ctx context.Context, t Target, serverPort string, opts RadiusServerSetOptions) error {
	args := []string{serverPort}
	if opts.Secret != "" {
		args = append(args, "/SECRET:"+opts.Secret)
	}
	if opts.RetryInterval != "" {
		args = append(args, "/RETRY_INTERVAL:"+opts.RetryInterval)
	}
	_, err := c.Run(ctx, t, "RadiusServerSet", args...)
	return err
}

func (c *Client) RadiusServerDelete(ctx context.Context, t Target) error {
	_, err := c.Run(ctx, t, "RadiusServerDelete")
	return err
}

func (c *Client) LogEnable(ctx context.Context, t Target, logType string) error {
	_, err := c.Run(ctx, t, "LogEnable", logType)
	return err
}

func (c *Client) LogDisable(ctx context.Context, t Target, logType string) error {
	_, err := c.Run(ctx, t, "LogDisable", logType)
	return err
}

func (c *Client) LogSwitchSet(ctx context.Context, t Target, logType, switchCycle string) error {
	_, err := c.Run(ctx, t, "LogSwitchSet", logType, "/SWITCH:"+switchCycle)
	return err
}

func (c *Client) LogPacketSaveType(ctx context.Context, t Target, packetType, saveType string) error {
	_, err := c.Run(ctx, t, "LogPacketSaveType", "/TYPE:"+packetType, "/SAVE:"+saveType)
	return err
}

// --- Hub Management Mode: access list (packet filter) ---

func (c *Client) AccessList(ctx context.Context, t Target) (Table, error) {
	out, err := c.Run(ctx, t, "AccessList")
	if err != nil {
		return Table{}, err
	}
	return ParseCSVTable(out)
}

// AccessAddOptions contains fields required to create a new Access List rule.
type AccessAddOptions struct {
	Pass        bool   // true = pass, false = discard
	Memo        string // description
	Priority    int    // rule priority (>= 1)
	SrcUser     string
	DstUser     string
	SrcMAC      string
	DstMAC      string
	SrcIP       string // e.g. "0.0.0.0/0"
	DstIP       string // e.g. "0.0.0.0/0"
	Protocol    string // e.g. "0" or "tcp", "udp", "icmpv4"
	SrcPort     string // e.g. "0"
	DstPort     string // e.g. "0"
	TcpState    string // e.g. "" or "Established" / "Unestablished"
}

func (c *Client) AccessAdd(ctx context.Context, t Target, opts AccessAddOptions) error {
	action := "discard"
	if opts.Pass {
		action = "pass"
	}
	prio := strconv.Itoa(opts.Priority)
	if opts.Priority <= 0 {
		prio = "100"
	}
	srcIP := opts.SrcIP
	if srcIP == "" {
		srcIP = "0.0.0.0/0"
	}
	dstIP := opts.DstIP
	if dstIP == "" {
		dstIP = "0.0.0.0/0"
	}
	proto := opts.Protocol
	if proto == "" {
		proto = "0"
	}
	srcPort := opts.SrcPort
	if srcPort == "" {
		srcPort = "0"
	}
	dstPort := opts.DstPort
	if dstPort == "" {
		dstPort = "0"
	}

	stdinLines := []string{
		action,
		opts.Memo,
		prio,
		opts.SrcUser,
		opts.DstUser,
		opts.SrcMAC,
		opts.DstMAC,
		srcIP,
		dstIP,
		proto,
		srcPort,
		dstPort,
		opts.TcpState,
	}

	_, err := c.RunWithInput(ctx, t, "AccessAdd", nil, stdinLines)
	return err
}

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

// CascadeCreateOptions specifies parameters to establish a new cascade connection.
type CascadeCreateOptions struct {
	Name       string
	ServerHost string
	ServerPort int
	Hub        string
	User       string
}

func (c *Client) CascadeCreate(ctx context.Context, t Target, opts CascadeCreateOptions) error {
	port := opts.ServerPort
	if port <= 0 {
		port = 443
	}
	serverAddr := fmt.Sprintf("%s:%d", opts.ServerHost, port)
	stdinLines := []string{
		opts.Name,
		serverAddr,
		opts.Hub,
		opts.User,
	}
	_, err := c.RunWithInput(ctx, t, "CascadeCreate", nil, stdinLines)
	return err
}

func (c *Client) CascadeDelete(ctx context.Context, t Target, name string) error {
	_, err := c.Run(ctx, t, "CascadeDelete", name)
	return err
}

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
