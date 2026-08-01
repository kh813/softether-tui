package ui

import "github.com/charmbracelet/lipgloss"

// confirmKind identifies which destructive action a confirmDialog is guarding.
type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmDeleteProfile
	confirmDeleteHub
	confirmDeleteUser
	confirmDeleteGroup
	confirmDisconnectSession
	confirmDeleteListener
	confirmDeleteAccessRule
	confirmDeleteCascade
	confirmDeleteBridge
	confirmDeleteAccount
	confirmToggleSecureNAT
	confirmToggleDHCP
	confirmEnableListener
	confirmDisableListener
	confirmRemoveGroupMembers
	confirmQuitUnsaved
	confirmQuitApp
)

// confirmDialog is a modal yes/no prompt for destructive actions. It stores
// the action as a kind+target pair rather than a closure: Model.Update uses
// value semantics (a fresh Model each call), so a closure captured at Show
// time would see stale profile state by the time the user answers.
type confirmDialog struct {
	active  bool
	kind    confirmKind
	target  string
	message string
}

func (c *confirmDialog) Show(kind confirmKind, target, message string) {
	c.active = true
	c.kind = kind
	c.target = target
	c.message = message
}

func (c *confirmDialog) Hide() {
	*c = confirmDialog{}
}

func (c *confirmDialog) View() string {
	content := tr("確認") + "\n\n" + c.message + "\n\n" + tr("[ y: 実行する ]   [ n: キャンセル ]")
	return borderStyle().BorderForeground(lipgloss.Color("196")).Render(content)
}
