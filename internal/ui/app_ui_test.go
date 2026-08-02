package ui

import (
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
