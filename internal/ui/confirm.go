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
	confirmDiscardChanges
)

// confirmDialog is a modal yes/no prompt for destructive actions. It stores
// the action as a kind+target pair rather than a closure: Model.Update uses
// value semantics (a fresh Model each call), so a closure captured at Show
// time would see stale profile state by the time the user answers.
type confirmButton int

const (
	confirmBtnYes confirmButton = iota
	confirmBtnNo
)

type confirmDialog struct {
	active  bool
	kind    confirmKind
	target  string
	message string
	focus   confirmButton
}

func (c *confirmDialog) Show(kind confirmKind, target, message string) {
	c.active = true
	c.kind = kind
	c.target = target
	c.message = message
	c.focus = confirmBtnYes
}

func (c *confirmDialog) Hide() {
	*c = confirmDialog{}
}

func (c *confirmDialog) View() string {
	yesStyle := inactiveTabStyle
	noStyle := inactiveTabStyle
	yesMarker := "  "
	noMarker := "  "

	if c.focus == confirmBtnYes {
		yesStyle = saveKeyStyle
		yesMarker = "> "
	} else {
		noStyle = saveKeyStyle
		noMarker = "> "
	}

	yesBtn := yesMarker + yesStyle.Render(tr(" [ y: OK ] "))
	noBtn := noMarker + noStyle.Render(tr(" [ n: キャンセル ] "))

	content := tr("確認") + "\n\n" + c.message + "\n\n" + yesBtn + "   " + noBtn
	return borderStyle().BorderForeground(lipgloss.Color("196")).Render(content)
}
