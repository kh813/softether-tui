package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
)

type formField int

const (
	fieldName formField = iota
	fieldHost
	fieldPort
	fieldHub
	fieldMode
	fieldSave
	fieldCount
)

var formInputCount = int(fieldMode) // number of textinput.Model fields (excludes fieldMode and fieldSave)

var profileModeOrder = []config.Mode{config.ModeServer, config.ModeBridge, config.ModeClient}

// profileForm is the add/edit screen for a connection profile.
type profileForm struct {
	inputs         [4]textinput.Model // name, host, port, hub
	mode           config.Mode
	focus          formField
	editing        bool   // true when editing an existing profile
	original       string // original name, to detect renames on save
	dropdownActive bool
	dropdownCursor int
}

func newProfileForm() *profileForm {
	f := &profileForm{mode: config.ModeServer}
	placeholders := []string{tr("表示名"), tr("ホスト"), "443", tr("Hub (任意)")}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	f.inputs[2].SetValue("443")
	f.setFocus(fieldName)
	return f
}

// LoadProfile populates the form for editing an existing profile.
func (f *profileForm) LoadProfile(p config.Profile) {
	f.inputs[0].SetValue(p.Name)
	f.inputs[1].SetValue(p.Host)
	f.inputs[2].SetValue(strconv.Itoa(p.Port))
	f.inputs[3].SetValue(p.Hub)
	f.mode = p.Mode
	f.editing = true
	f.original = p.Name
	f.setFocus(fieldName)
}

// Reset clears the form for creating a new profile.
func (f *profileForm) Reset() {
	f.inputs[0].SetValue("")
	f.inputs[1].SetValue("")
	f.inputs[2].SetValue("443")
	f.inputs[3].SetValue("")
	f.mode = config.ModeServer
	f.editing = false
	f.original = ""
	f.setFocus(fieldName)
}

func (f *profileForm) IsDirty() bool {
	if !f.editing {
		return strings.TrimSpace(f.inputs[0].Value()) != "" ||
			strings.TrimSpace(f.inputs[1].Value()) != "" ||
			strings.TrimSpace(f.inputs[3].Value()) != ""
	}
	return false
}

func (f *profileForm) setFocus(field formField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	if int(field) < len(f.inputs) {
		f.inputs[field].Focus()
	}
}

// Build validates the current form state and returns the resulting profile.
func (f *profileForm) Build() (config.Profile, error) {
	name := strings.TrimSpace(f.inputs[fieldName].Value())
	host := strings.TrimSpace(f.inputs[fieldHost].Value())
	portStr := strings.TrimSpace(f.inputs[fieldPort].Value())
	hub := strings.TrimSpace(f.inputs[fieldHub].Value())

	if name == "" {
		return config.Profile{}, errors.New(tr("表示名は必須です"))
	}
	if host == "" {
		return config.Profile{}, errors.New(tr("ホストは必須です"))
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return config.Profile{}, fmt.Errorf(tr("ポート番号が不正です: %q"), portStr)
	}

	return config.Profile{
		Name: name,
		Host: host,
		Port: port,
		Mode: f.mode,
		Hub:  hub,
	}, nil
}

func (f *profileForm) Update(msg tea.KeyMsg) tea.Cmd {
	if f.dropdownActive {
		switch msg.String() {
		case "up", "k":
			if f.dropdownCursor > 0 {
				f.dropdownCursor--
			}
			return nil
		case "down", "j":
			if f.dropdownCursor < len(profileModeOrder)-1 {
				f.dropdownCursor++
			}
			return nil
		case "enter":
			f.mode = profileModeOrder[f.dropdownCursor]
			f.dropdownActive = false
			return nil
		case "esc":
			f.dropdownActive = false
			return nil
		}
		return nil
	}

	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % fieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + fieldCount) % fieldCount)
		return nil
	case "left", "right":
		if f.focus == fieldMode {
			f.cycleMode(msg.String() == "right")
		}
		return nil
	case "enter":
		if f.focus == fieldMode {
			f.dropdownActive = true
			for i, m := range profileModeOrder {
				if m == f.mode {
					f.dropdownCursor = i
					break
				}
			}
			return nil
		}
	}

	if int(f.focus) < formInputCount {
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
	return nil
}

func (f *profileForm) cycleMode(forward bool) {
	idx := 0
	for i, m := range profileModeOrder {
		if m == f.mode {
			idx = i
		}
	}
	if forward {
		idx = (idx + 1) % len(profileModeOrder)
	} else {
		idx = (idx - 1 + len(profileModeOrder)) % len(profileModeOrder)
	}
	f.mode = profileModeOrder[idx]
}

func (f *profileForm) View() string {
	labels := []string{tr("接続名"), tr("ホスト"), tr("ポート"), tr("Hub (任意)")}
	modeLabelStr := tr("モード")

	// Calculate maximum label string length dynamically so colons align perfectly in both JA and EN
	maxLen := len(modeLabelStr)
	for _, l := range labels {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	maxLen += 1 // account for colon

	var b strings.Builder

	title := tr("プロファイル追加")
	if f.editing {
		title = tr("プロファイル編集: ") + f.original
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == formField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-*s %s\n", marker, maxLen, labels[i]+":", in.View())
	}

	modeMarker := "  "
	modeStyle := statusBarStyle
	if f.focus == fieldMode {
		modeMarker = "> "
		modeStyle = selectedStyle
	}
	fmt.Fprintf(&b, "%s%-*s < %s >\n", modeMarker, maxLen, modeLabelStr+":", modeStyle.Render(modeLabel(f.mode)))

	if f.dropdownActive {
		b.WriteString("\n" + f.renderModeDropdown())
	} else {
		nameValid := strings.TrimSpace(f.inputs[fieldName].Value()) != ""
		hostValid := strings.TrimSpace(f.inputs[fieldHost].Value()) != ""
		portValid := strings.TrimSpace(f.inputs[fieldPort].Value()) != ""
		canSave := nameValid && hostValid && portValid

		b.WriteString("\n")
		saveMarker := "  "
		if f.focus == fieldSave {
			saveMarker = "> "
		}

		if canSave {
			if f.focus == fieldSave {
				b.WriteString(saveMarker + saveKeyStyle.Render(" [ Save ] ") + "\n")
			} else {
				b.WriteString(saveMarker + inactiveTabStyle.Render(" [ Save ] ") + "\n")
			}
			b.WriteString("\n" + renderHelp(
				"Tab/↑↓", tr("項目移動"),
				"Enter", tr("保存 (Save)"),
				"←/→", tr("モード切替"),
				"Esc", tr("キャンセル"),
			))
		} else {
			b.WriteString(saveMarker + dimStyle.Render("[ Save - Please fill required fields ]") + "\n")
			b.WriteString("\n" + renderHelp(
				"Tab/↑↓", tr("項目移動"),
				"←/→", tr("モード切替"),
				"Esc", tr("キャンセル"),
			))
		}
	}
	return b.String()
}

func (f *profileForm) renderModeDropdown() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr("--- モード選択 (↑/↓:移動 Enter:決定 Esc:閉じる) ---")) + "\n")
	for i, m := range profileModeOrder {
		marker := "  "
		style := statusBarStyle
		if f.dropdownCursor == i {
			marker = "> "
			style = selectedStyle
		}
		fmt.Fprintf(&b, "%s%s\n", marker, style.Render(modeLabel(m)))
	}
	return b.String()
}
