package vpncmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"softether-tui/internal/config"
)

func TestBuildArgs(t *testing.T) {
	got := buildArgs(Target{Host: "vpn.example.jp", Port: 443, Mode: config.ModeServer, Hub: "Main"}, "UserList", []string{"/EXTRA"})
	want := []string{"vpn.example.jp:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "UserList", "/EXTRA"}
	assertEqualSlice(t, got, want)
}

func TestBuildArgsBridgeModeNoHub(t *testing.T) {
	got := buildArgs(Target{Host: "10.0.2.5", Port: 5555, Mode: config.ModeBridge}, "BridgeList", nil)
	want := []string{"10.0.2.5:5555", "/BRIDGE", "/CSV", "/CMD", "BridgeList"}
	assertEqualSlice(t, got, want)
}

func TestBuildArgsClientModeOmitsHub(t *testing.T) {
	// Even if a Target carries a Hub (e.g. a leftover profile default),
	// /CLIENT mode commands must never emit /HUB: - accounts aren't hub-scoped.
	target := Target{Host: "127.0.0.1", Port: 5555, Mode: config.ModeClient, Hub: "Main"}
	got := buildArgs(target, "AccountList", nil)
	want := []string{"127.0.0.1:5555", "/CLIENT", "/CSV", "/CMD", "AccountList"}
	assertEqualSlice(t, got, want)
}

func TestAccountCreateArgs(t *testing.T) {
	target := Target{Host: "127.0.0.1", Port: 5555, Mode: config.ModeClient}
	got := buildArgs(target, "AccountCreate", []string{"work", "/SERVER:vpn.example.jp:443", "/HUB:Main"})
	want := []string{
		"127.0.0.1:5555", "/CLIENT", "/CSV", "/CMD", "AccountCreate",
		"work", "/SERVER:vpn.example.jp:443", "/HUB:Main",
	}
	assertEqualSlice(t, got, want)
}

func assertEqualSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d: got %q, want %q (full got=%v)", i, got[i], want[i], got)
		}
	}
}

func TestParseCSV(t *testing.T) {
	const sample = "Item,Value\r\n" +
		"Server Type,Standalone Server\r\n" +
		"Version,SoftEther VPN Server (test)\r\n"

	kv, err := ParseCSV(sample)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if kv["Server Type"] != "Standalone Server" {
		t.Errorf("Server Type = %q", kv["Server Type"])
	}
	if kv["Version"] != "SoftEther VPN Server (test)" {
		t.Errorf("Version = %q", kv["Version"])
	}
}

func TestParseCSVTable(t *testing.T) {
	const sample = "HubName,Online,Sessions,Users,Groups,Type\r\n" +
		"Main,Yes,18,12,3,Standalone\r\n" +
		"Guest,Yes,9,5,1,Standalone\r\n" +
		"Staging,No,0,3,0,Standalone\r\n"

	table, err := ParseCSVTable(sample)
	if err != nil {
		t.Fatalf("ParseCSVTable: %v", err)
	}
	if len(table.Rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(table.Rows))
	}
	if table.Rows[0]["HubName"] != "Main" {
		t.Errorf("row 0 HubName = %q", table.Rows[0]["HubName"])
	}
	if got := table.NameOf(table.Rows[2]); got != "Staging" {
		t.Errorf("NameOf(row 2) = %q, want Staging", got)
	}
}

func TestParseCSVTableEmpty(t *testing.T) {
	table, err := ParseCSVTable("")
	if err != nil {
		t.Fatalf("ParseCSVTable: %v", err)
	}
	if len(table.Rows) != 0 || len(table.Headers) != 0 {
		t.Fatalf("got %+v, want empty table", table)
	}
}

func TestHubListUsesServerAdminScope(t *testing.T) {
	// HubList must clear any Hub set on the Target: it lists all hubs from
	// Server Admin Mode, it does not operate within a single selected hub.
	got := buildArgs(hubScoped(Target{Host: "localhost", Port: 443, Hub: "Main"}), "HubList", nil)
	for _, a := range got {
		if strings.HasPrefix(a, "/HUB:") {
			t.Fatalf("buildArgs included %q, want no /HUB: flag for a hub-scoped(cleared) target", a)
		}
	}
}

func TestHubSetOnlineScopesIntoHub(t *testing.T) {
	target := Target{Host: "localhost", Port: 443}
	target.Hub = "Main"
	got := buildArgs(target, "Online", nil)
	want := []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "Online"}
	assertEqualSlice(t, got, want)
}

func TestWithHubScopesUserAndGroupCommands(t *testing.T) {
	base := Target{Host: "localhost", Port: 443}
	scoped := base.WithHub("Main")

	got := buildArgs(scoped, "UserList", nil)
	want := []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "UserList"}
	assertEqualSlice(t, got, want)

	if base.Hub != "" {
		t.Fatalf("WithHub mutated the original Target: Hub=%q", base.Hub)
	}
}

func TestUserCreateArgsIncludeOptionalFields(t *testing.T) {
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")
	got := buildArgs(target, "UserCreate", []string{"alice", "/GROUP:admins", "/REALNAME:Alice", "/NOTE:test"})
	want := []string{
		"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "UserCreate",
		"alice", "/GROUP:admins", "/REALNAME:Alice", "/NOTE:test",
	}
	assertEqualSlice(t, got, want)
}

func TestSessionAndLogCommandsScopeIntoHub(t *testing.T) {
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")

	got := buildArgs(target, "SessionDisconnect", []string{"SID-ABC"})
	want := []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "SessionDisconnect", "SID-ABC"}
	assertEqualSlice(t, got, want)

	got = buildArgs(target, "LogGet", nil)
	want = []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "LogGet"}
	assertEqualSlice(t, got, want)
}

func TestListenerCommandsUseServerAdminScope(t *testing.T) {
	// Listeners are server-wide, so even if the Target carries a Hub (e.g.
	// from a profile default), listener commands must clear it.
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")

	got := buildArgs(hubScoped(target), "ListenerCreate", []string{"1194"})
	want := []string{"localhost:443", "/SERVER", "/CSV", "/CMD", "ListenerCreate", "1194"}
	assertEqualSlice(t, got, want)
}

func TestAccessAndSecureNatCommandsScopeIntoHub(t *testing.T) {
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")

	got := buildArgs(target, "AccessDelete", []string{"3"})
	want := []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "AccessDelete", "3"}
	assertEqualSlice(t, got, want)

	got = buildArgs(target, "SecureNatEnable", nil)
	want = []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "SecureNatEnable"}
	assertEqualSlice(t, got, want)
}

func TestCascadeCommandsScopeIntoHub(t *testing.T) {
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")

	got := buildArgs(target, "CascadeOnline", []string{"ToOsaka"})
	want := []string{"localhost:443", "/SERVER", "/HUB:Main", "/CSV", "/CMD", "CascadeOnline", "ToOsaka"}
	assertEqualSlice(t, got, want)
}

func TestBridgeCommandsUseServerAdminScopeAndBridgeMode(t *testing.T) {
	// BridgeList/Create/Delete are server-wide, so any Hub on the Target
	// must be cleared, regardless of whether the profile is /SERVER or
	// /BRIDGE mode (VPN Bridge is the primary user of local bridges).
	target := Target{Host: "10.0.2.5", Port: 5555, Mode: config.ModeBridge}.WithHub("Main")

	got := buildArgs(hubScoped(target), "BridgeCreate", []string{"Main", "/DEVICE:tap_soft", "/TAP:yes"})
	want := []string{"10.0.2.5:5555", "/BRIDGE", "/CSV", "/CMD", "BridgeCreate", "Main", "/DEVICE:tap_soft", "/TAP:yes"}
	assertEqualSlice(t, got, want)
}

func TestRunWithoutBinaryReturnsErrNotFound(t *testing.T) {
	c := NewClient("")
	_, err := c.Run(context.Background(), Target{Host: "localhost", Port: 443}, "ServerInfoGet")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

// TestRunEndToEnd exercises the full Run() path (argument building, stdin
// password delivery, stdout capture) against a fake vpncmd shell script,
// since a real vpncmd binary/server is not available in this environment.
func TestRunEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake vpncmd script requires a POSIX shell")
	}

	dir := t.TempDir()
	stdinCapture := filepath.Join(dir, "stdin.txt")
	script := "#!/bin/sh\n" +
		"cat > \"" + stdinCapture + "\"\n" +
		"echo 'Item,Value'\n" +
		"echo 'Server Type,Standalone Server'\n"

	fakeBin := filepath.Join(dir, "vpncmd")
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake vpncmd: %v", err)
	}

	c := NewClient(fakeBin)
	kv, err := c.ServerInfo(context.Background(), Target{
		Host:     "localhost",
		Port:     443,
		Mode:     config.ModeServer,
		Password: "s3cret",
	})
	if err != nil {
		t.Fatalf("ServerInfo: %v", err)
	}
	if kv["Server Type"] != "Standalone Server" {
		t.Errorf("Server Type = %q", kv["Server Type"])
	}

	stdin, err := os.ReadFile(stdinCapture)
	if err != nil {
		t.Fatalf("read stdin capture: %v", err)
	}
	if string(stdin) != "s3cret\n" {
		t.Errorf("stdin = %q, want %q", string(stdin), "s3cret\n")
	}
}

// TestUserExpiresSetFormatsDate captures the argv a fake vpncmd receives to
// confirm the expiration date is formatted the way SoftEther's CLI expects.
func TestUserExpiresSetFormatsDate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake vpncmd script requires a POSIX shell")
	}

	dir := t.TempDir()
	argvCapture := filepath.Join(dir, "argv.txt")
	script := "#!/bin/sh\n" +
		"cat /dev/null > /dev/null\n" + // consume nothing; drain stdin below
		"cat > /dev/null\n" +
		"printf '%s\\n' \"$@\" > \"" + argvCapture + "\"\n"

	fakeBin := filepath.Join(dir, "vpncmd")
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake vpncmd: %v", err)
	}

	c := NewClient(fakeBin)
	target := Target{Host: "localhost", Port: 443}.WithHub("Main")
	expires := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
	if err := c.UserExpiresSet(context.Background(), target, "tanaka", expires); err != nil {
		t.Fatalf("UserExpiresSet: %v", err)
	}

	argv, err := os.ReadFile(argvCapture)
	if err != nil {
		t.Fatalf("read argv capture: %v", err)
	}
	got := strings.TrimSpace(string(argv))
	want := "tanaka\n2026/12/31 23:59:00"
	if !strings.HasSuffix(got, want) {
		t.Errorf("argv = %q, want suffix %q", got, want)
	}
}

func TestLocateSameDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake vpncmd script requires a POSIX shell")
	}

	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "vpncmd")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake vpncmd: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	loc, err := Locate()
	if err != nil {
		t.Fatalf("Locate failed: %v", err)
	}

	expected, _ := filepath.Abs(fakeBin)
	locEval, _ := filepath.EvalSymlinks(loc)
	expectedEval, _ := filepath.EvalSymlinks(expected)
	if locEval != expectedEval {
		t.Errorf("Locate() = %q, want %q", locEval, expectedEval)
	}
}
