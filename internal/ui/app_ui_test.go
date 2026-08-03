package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/vpncmd"
)

func sendKey(m Model, key string) Model {
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "esc":
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	}
	updated, _ := m.Update(keyMsg)
	return updated.(Model)
}

func sendRuneKey(m Model, r rune) Model {
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
	updated, _ := m.Update(keyMsg)
	return updated.(Model)
}

func setupTestModel(t *testing.T) Model {
	store := config.NewStore(t.TempDir() + "/profiles.yaml")
	client := vpncmd.NewClient("vpncmd")
	return New(store, client, "v0.0.1-test")
}

func TestUINavigationAndCreationKeyC(t *testing.T) {
	m := setupTestModel(t)
	// Initial screen should be screenProfileList
	if m.screen != screenProfileList {
		t.Fatalf("expected screenProfileList, got %v", m.screen)
	}

	// Pressing 'c' should open screenProfileForm
	m = sendKey(m, "c")
	if m.screen != screenProfileForm {
		t.Fatalf("expected screenProfileForm on 'c' key, got %v", m.screen)
	}

	// Pressing 'esc' in form should return to screenProfileList
	m = sendKey(m, "esc")
	if m.screen != screenProfileList {
		t.Fatalf("expected screenProfileList on 'esc', got %v", m.screen)
	}
}

func TestProfileFormSaveButtonFocus(t *testing.T) {
	m := setupTestModel(t)
	m = sendKey(m, "c")

	// Initially focused on field 0 (fieldName)
	if m.form.focus != fieldName {
		t.Fatalf("expected focus fieldName, got %v", m.form.focus)
	}

	// Press Down 4 times to navigate fieldName -> fieldHost -> fieldPort -> fieldHub -> fieldMode -> fieldSave
	m = sendKey(m, "down") // fieldHost
	m = sendKey(m, "down") // fieldPort
	m = sendKey(m, "down") // fieldHub
	m = sendKey(m, "down") // fieldMode
	m = sendKey(m, "down") // fieldSave

	if m.form.focus != fieldSave {
		t.Fatalf("expected focus fieldSave after pressing Down from fieldMode, got %v", m.form.focus)
	}
}

func TestFullwidthIMEKeyNormalization(t *testing.T) {
	m := setupTestModel(t)

	// Pressing fullwidth 'ｃ' (Ｕ+ＦＦ43) should trigger 'c' and open screenProfileForm
	m = sendRuneKey(m, 'ｃ')
	if m.screen != screenProfileForm {
		t.Fatalf("expected fullwidth 'ｃ' to trigger creation form, got %v", m.screen)
	}

	m = sendKey(m, "esc")
	if m.screen != screenProfileList {
		t.Fatalf("expected return to profile list, got %v", m.screen)
	}
}

func TestBackspaceDoesNotNavigateBack(t *testing.T) {
	m := setupTestModel(t)
	// Navigate into dashboard
	m.profiles = []config.Profile{{Name: "Test", Host: "127.0.0.1", Port: 443}}
	m.screen = screenDashboard

	// Send backspace
	m = sendKey(m, "backspace")

	// Verify screen did NOT change to screenProfileList
	if m.screen != screenDashboard {
		t.Fatalf("backspace caused unexpected screen navigation from dashboard: got %v", m.screen)
	}
}

func TestTwoPhaseSecureNATEscEditing(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenHubDetail
	m.hubDetail.tab = hubTabSecureNAT
	m.hubDetail.secureNatEditing = true
	m.hubDetail.secureNatEditingField = fieldNatIP

	// Pressing Esc during field editing should exit editing mode without changing screen
	m = sendKey(m, "esc")
	if m.hubDetail.secureNatEditing {
		t.Fatalf("expected secureNatEditing to become false on Esc")
	}
	if m.screen != screenHubDetail {
		t.Fatalf("expected screen to remain screenHubDetail, got %v", m.screen)
	}

	// Now in field navigation mode with dirty = true, Esc should open confirmDiscardChanges modal
	m.hubDetail.secureNatDirty = true
	m = sendKey(m, "esc")
	if !m.confirm.active || m.confirm.kind != confirmDiscardChanges {
		t.Fatalf("expected confirmDiscardChanges modal on Esc when dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	// Cancel modal
	m = sendKey(m, "n")

	// Tab key when dirty should ALSO trigger confirmDiscardChanges modal
	m = sendKey(m, "tab")
	if !m.confirm.active || m.confirm.kind != confirmDiscardChanges {
		t.Fatalf("expected confirmDiscardChanges modal on Tab key when dirty")
	}
}

func TestUnsavedChangesModalGuard(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenUserDetail
	m.userDetail.dirty = true

	// Pressing Esc when dirty should trigger confirmDiscardChanges modal
	m = sendKey(m, "esc")
	if !m.confirm.active || m.confirm.kind != confirmDiscardChanges {
		t.Fatalf("expected confirmDiscardChanges modal for UserDetail when dirty")
	}

	// Cancelling modal (n) should leave user on UserDetail
	m = sendKey(m, "n")
	if m.confirm.active {
		t.Fatalf("expected confirm modal to close on 'n'")
	}
	if m.screen != screenUserDetail {
		t.Fatalf("expected screen to remain screenUserDetail after cancelling discard, got %v", m.screen)
	}
}

func TestCascadeFormEscProtectionAndButtonFocus(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenCascadeForm
	m.cascadeForm.Reset()

	// Type something in Setting Name
	m = sendRuneKey(m, 'X')

	// Pressing Esc with inputs should trigger confirmDiscardChanges modal
	m = sendKey(m, "esc")
	if !m.confirm.active || m.confirm.kind != confirmDiscardChanges {
		t.Fatalf("expected confirmDiscardChanges modal on Esc when cascadeForm is dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	// Cancel modal
	m = sendKey(m, "n")

	// Verify focus navigation reaches fieldSave then fieldTest
	for i := 0; i < 6; i++ {
		m = sendKey(m, "down")
	}
	if m.cascadeForm.focus != cascadeFieldSave {
		t.Fatalf("expected focus cascadeFieldSave, got %v", m.cascadeForm.focus)
	}

	m = sendKey(m, "down")
	if m.cascadeForm.focus != cascadeFieldTest {
		t.Fatalf("expected focus cascadeFieldTest, got %v", m.cascadeForm.focus)
	}
}

func TestACLFormProtocolSelectionAndEscProtection(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenACLForm
	m.aclForm.Reset()

	// Initial protocol should be ALL (0)
	if aclProtocolOrder[m.aclForm.protocolIdx].Val != "0" {
		t.Fatalf("expected default protocol '0', got %v", aclProtocolOrder[m.aclForm.protocolIdx].Val)
	}

	// Move focus to Protocol (4 down keypresses)
	for i := 0; i < 4; i++ {
		m = sendKey(m, "down")
	}
	if m.aclForm.focus != aclFieldProtocol {
		t.Fatalf("expected focus aclFieldProtocol, got %v", m.aclForm.focus)
	}

	// Press right arrow to switch protocol to ICMPv4 (1)
	m = sendKey(m, "right")
	if aclProtocolOrder[m.aclForm.protocolIdx].Val != "1" {
		t.Fatalf("expected protocol ICMPv4 ('1'), got %v", aclProtocolOrder[m.aclForm.protocolIdx].Val)
	}

	// Verify dirty state flag is set by selection change
	if !m.aclForm.IsDirty() {
		t.Fatalf("expected aclForm to be dirty after protocol selection change")
	}

	// Press Esc -> confirmDiscardChanges modal must pop up
	m = sendKey(m, "esc")
	if !m.confirm.active || m.confirm.kind != confirmDiscardChanges {
		t.Fatalf("expected confirmDiscardChanges modal on Esc when aclForm is dirty")
	}
}

func TestACLFormEnterKeySelectFieldsToggling(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenACLForm
	m.aclForm.Reset()

	// Initial focus on Action (aclFieldPass), pass = true
	if !m.aclForm.pass {
		t.Fatalf("expected initial pass = true")
	}
	// Pressing Enter on Action toggles pass -> false (Discard)
	m = sendKey(m, "enter")
	if m.aclForm.pass {
		t.Fatalf("expected pass = false after pressing Enter on Action field")
	}

	// Move focus to Status (aclFieldEnable)
	m = sendKey(m, "down")
	if m.aclForm.focus != aclFieldEnable {
		t.Fatalf("expected focus aclFieldEnable, got %v", m.aclForm.focus)
	}
	if !m.aclForm.enable {
		t.Fatalf("expected initial enable = true")
	}
	// Pressing Enter on Status toggles enable -> false (Disable)
	m = sendKey(m, "enter")
	if m.aclForm.enable {
		t.Fatalf("expected enable = false after pressing Enter on Status field")
	}
}

func TestGroupDetailAddMemberPrompt(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenGroupDetail
	m.groupDetail = groupDetailState{
		groupName: "Group1",
		members:   []string{"User03"},
		allUsers:  []string{"User01", "User02", "User03"},
	}

	// Pressing 'a' on Group Detail when not dirty should trigger promptAddGroupMember
	m = sendKey(m, "a")
	if !m.prompt.active || m.prompt.kind != promptAddGroupMember {
		t.Fatalf("expected promptAddGroupMember on 'a' key in Group Detail")
	}
	m.prompt.Hide()

	// 'c' is kept as an alias for 'a' (Create-family shortcut) when not dirty.
	m = sendKey(m, "c")
	if !m.prompt.active || m.prompt.kind != promptAddGroupMember {
		t.Fatalf("expected promptAddGroupMember on 'c' key in Group Detail when not dirty")
	}
	m.prompt.Hide()

	// Move focus to User01 row (cursor = 3) and press Space -> stages change, sets dirty = true
	m.groupDetail.cursor = 3
	m = sendKey(m, " ")
	if !m.groupDetail.dirty {
		t.Fatalf("expected groupDetail to be dirty after Space key on user row")
	}
	if !m.groupDetail.isMember("User01") {
		t.Fatalf("expected User01 to be staged as member")
	}

	// While dirty, 'c' must no longer discard (it collided with the app-wide
	// "c" = Create shortcut) and must not open the add-member prompt either.
	m = sendKey(m, "c")
	if !m.groupDetail.dirty {
		t.Fatalf("expected 'c' to have no discard effect while groupDetail is dirty")
	}
	if m.prompt.active {
		t.Fatalf("expected 'c' to not open the add-member prompt while groupDetail is dirty")
	}

	// 'n' opens the discard confirmation instead of discarding immediately.
	m = sendKey(m, "n")
	if !m.confirm.active || m.confirm.kind != confirmDiscardInPlace {
		t.Fatalf("expected confirmDiscardInPlace modal on 'n' when dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	// Confirming with 'y' discards the pending member edit in place.
	m = sendKey(m, "y")
	if m.groupDetail.dirty {
		t.Fatalf("expected groupDetail dirty to be false after confirming discard")
	}
	if m.groupDetail.isMember("User01") {
		t.Fatalf("expected User01 staged edit to be cleared on discard")
	}
	if m.screen != screenGroupDetail {
		t.Fatalf("expected screen to remain screenGroupDetail, got %v", m.screen)
	}
}

// TestSecureNATDetailDiscardKeyIsNNotC guards against the historical key
// collision where 'c' meant "discard changes" here but "Create" everywhere
// else in the app. 'n' (matching the confirm dialog's own "n: Cancel"
// convention) must now require confirmation via confirmDiscardInPlace and
// must reset dirty/editedValues without leaving screenSecureNATDetail.
func TestSecureNATDetailDiscardKeyIsNNotC(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenSecureNATDetail
	m.secureNatDetail.dirty = true
	m.secureNatDetail.editedValues = map[editableSecureNATField]string{fieldNatIP: "10.0.0.1"}

	// 'c' must no longer discard - it has no effect on this screen anymore.
	m = sendKey(m, "c")
	if !m.secureNatDetail.dirty {
		t.Fatalf("expected 'c' to no longer discard changes on SecureNAT detail screen")
	}

	// 'n' must open the confirmation dialog rather than discarding immediately.
	m = sendKey(m, "n")
	if !m.confirm.active || m.confirm.kind != confirmDiscardInPlace {
		t.Fatalf("expected confirmDiscardInPlace modal on 'n' when dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	// Confirming with 'y' discards in place and stays on the same screen.
	m = sendKey(m, "y")
	if m.secureNatDetail.dirty {
		t.Fatalf("expected dirty to be false after confirming discard")
	}
	if len(m.secureNatDetail.editedValues) != 0 {
		t.Fatalf("expected editedValues to be cleared after confirming discard")
	}
	if m.screen != screenSecureNATDetail {
		t.Fatalf("expected screen to remain screenSecureNATDetail, got %v", m.screen)
	}
}

// TestUserDetailDiscardKeyIsNNotC mirrors TestSecureNATDetailDiscardKeyIsNNotC
// for the User Detail screen: 'c' must no longer discard changes, and 'n'
// must require confirmation via confirmDiscardInPlace before discarding,
// without navigating away from screenUserDetail.
func TestUserDetailDiscardKeyIsNNotC(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenUserDetail
	m.userDetail.dirty = true
	m.userDetail.editedValues = map[editableUserField]string{fieldRealName: "Someone"}

	m = sendKey(m, "c")
	if !m.userDetail.dirty {
		t.Fatalf("expected 'c' to no longer discard changes on User Detail screen")
	}

	m = sendKey(m, "n")
	if !m.confirm.active || m.confirm.kind != confirmDiscardInPlace {
		t.Fatalf("expected confirmDiscardInPlace modal on 'n' when dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	m = sendKey(m, "y")
	if m.userDetail.dirty {
		t.Fatalf("expected dirty to be false after confirming discard")
	}
	if len(m.userDetail.editedValues) != 0 {
		t.Fatalf("expected editedValues to be cleared after confirming discard")
	}
	if m.screen != screenUserDetail {
		t.Fatalf("expected screen to remain screenUserDetail, got %v", m.screen)
	}
}

// TestHubDetailSecureNATTabDiscardKeyIsNNotC mirrors the other three
// discard-key fixes for the SecureNAT tab embedded in the Hub Detail screen.
func TestHubDetailSecureNATTabDiscardKeyIsNNotC(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenHubDetail
	m.hubDetail.tab = hubTabSecureNAT
	m.hubDetail.secureNatDirty = true
	m.hubDetail.secureNatEditedValues = map[editableSecureNATField]string{fieldNatIP: "10.0.0.1"}

	m = sendKey(m, "c")
	if !m.hubDetail.secureNatDirty {
		t.Fatalf("expected 'c' to no longer discard changes on the Hub Detail SecureNAT tab")
	}

	m = sendKey(m, "n")
	if !m.confirm.active || m.confirm.kind != confirmDiscardInPlace {
		t.Fatalf("expected confirmDiscardInPlace modal on 'n' when dirty, active: %v, kind: %v", m.confirm.active, m.confirm.kind)
	}

	m = sendKey(m, "y")
	if m.hubDetail.secureNatDirty {
		t.Fatalf("expected secureNatDirty to be false after confirming discard")
	}
	if len(m.hubDetail.secureNatEditedValues) != 0 {
		t.Fatalf("expected secureNatEditedValues to be cleared after confirming discard")
	}
	if m.screen != screenHubDetail || m.hubDetail.tab != hubTabSecureNAT {
		t.Fatalf("expected to remain on screenHubDetail/hubTabSecureNAT, got screen=%v tab=%v", m.screen, m.hubDetail.tab)
	}
}

// TestAccountFormAndBridgeFormUseSharedHelpRenderer guards against the two
// forms falling back to a hand-rolled dimStyle.Render(...) help line: that
// path never highlights key names, unlike every other screen's renderHelp()
// output. renderHelp joins "key"+":"+"desc" (no space before the colon),
// while the old ad-hoc line used "key: desc" (space before the colon) -
// that's a stable, ANSI-independent way to tell the two apart under `go
// test` (no tty, so lipgloss emits no color codes either way).
func TestAccountFormAndBridgeFormUseSharedHelpRenderer(t *testing.T) {
	want := "Esc:" + tr("キャンセル")
	unwant := "Esc: " + tr("キャンセル")

	af := newAccountForm()
	aOut := af.View()
	if !strings.Contains(aOut, want) {
		t.Fatalf("expected accountForm help line to use renderHelp() key:desc format (%q), got: %q", want, aOut)
	}
	if strings.Contains(aOut, unwant) {
		t.Fatalf("accountForm help line still uses the old dimStyle.Render(\"key: desc\") format: %q", aOut)
	}

	bf := newBridgeForm()
	bOut := bf.View()
	if !strings.Contains(bOut, want) {
		t.Fatalf("expected bridgeForm help line to use renderHelp() key:desc format (%q), got: %q", want, bOut)
	}
	if strings.Contains(bOut, unwant) {
		t.Fatalf("bridgeForm help line still uses the old dimStyle.Render(\"key: desc\") format: %q", bOut)
	}
}

func TestSecureNATEditingEscProtection(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenHubDetail
	m.hubDetail.tab = hubTabSecureNAT
	m.hubDetail.secureNatEditing = true
	m.hubDetail.secureNatEditingField = fieldNatIP

	// Pressing esc while editing a field should cancel editing without changing screens
	m = sendKey(m, "esc")
	if m.hubDetail.secureNatEditing {
		t.Fatalf("expected secureNatEditing to be false after pressing esc while editing")
	}
	if m.screen != screenHubDetail {
		t.Fatalf("expected screen to remain screenHubDetail, got %v", m.screen)
	}
}

func TestSecureNatActionResultRefreshesServerInfo(t *testing.T) {
	m := setupTestModel(t)
	m.screen = screenHubDetail
	m.hubDetail.profile = config.Profile{Name: "localhost", Host: "127.0.0.1", Port: 443}
	m.hubDetail.hubName = "DEFAULT"

	// Trigger secureNatActionResultMsg
	updated, cmd := m.Update(secureNatActionResultMsg{action: "SecureNAT enabled", err: nil})
	m = updated.(Model)
	if cmd == nil {
		t.Fatalf("expected batch command containing fetchSecureNAT and fetchServerInfo")
	}
}

