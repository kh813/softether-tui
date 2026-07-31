package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/vpncmd"
)

type radiusFormField int

const (
	radiusFieldServerPort radiusFormField = iota
	radiusFieldSecret
	radiusFieldRetryInterval
	radiusFormFieldCount
)

type radiusForm struct {
	inputs [3]textinput.Model // server:port, secret, retry_interval
	focus  radiusFormField
}

func newRadiusForm() *radiusForm {
	serverPort := textinput.New()
	serverPort.Placeholder = tr("ホスト名:ポート (例: 192.168.1.100:1812)")
	serverPort.CharLimit = 128

	secret := textinput.New()
	secret.Placeholder = tr("共有シークレット (Secret)")
	secret.CharLimit = 128
	secret.EchoMode = textinput.EchoPassword
	secret.EchoCharacter = '*'

	retry := textinput.New()
	retry.Placeholder = tr("リトライ間隔 ms (任意例: 1000)")
	retry.CharLimit = 10

	f := &radiusForm{inputs: [3]textinput.Model{serverPort, secret, retry}}
	f.setFocus(radiusFieldServerPort)
	return f
}

func (f *radiusForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.setFocus(radiusFieldServerPort)
}

func (f *radiusForm) Load(info vpncmd.KeyValue) {
	f.Reset()
	if sp, ok := info["RADIUS Server Name"]; ok {
		f.inputs[radiusFieldServerPort].SetValue(sp)
	}
}

func (f *radiusForm) setFocus(field radiusFormField) {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = field
	f.inputs[field].Focus()
}

func (f *radiusForm) Build() (serverPort string, opts vpncmd.RadiusServerSetOptions, err error) {
	serverPort = strings.TrimSpace(f.inputs[radiusFieldServerPort].Value())
	if serverPort == "" {
		err = errors.New(tr("RADIUSサーバー (ホスト:ポート) は必須です"))
		return
	}
	secret := strings.TrimSpace(f.inputs[radiusFieldSecret].Value())
	if secret == "" {
		err = errors.New(tr("共有シークレットは必須です"))
		return
	}
	retry := strings.TrimSpace(f.inputs[radiusFieldRetryInterval].Value())
	opts = vpncmd.RadiusServerSetOptions{
		Secret:        secret,
		RetryInterval: retry,
	}
	return
}

func (f *radiusForm) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		f.setFocus((f.focus + 1) % radiusFormFieldCount)
		return nil
	case "shift+tab", "up":
		f.setFocus((f.focus - 1 + radiusFormFieldCount) % radiusFormFieldCount)
		return nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

func (f *radiusForm) View() string {
	labels := []string{tr("Server:Port"), tr("Shared Secret"), tr("Retry Interval (ms)")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(tr("RADIUSサーバー設定")) + "\n\n")

	for i, in := range f.inputs {
		marker := "  "
		if f.focus == radiusFormField(i) {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-22s %s\n", marker, labels[i]+":", in.View())
	}

	serverValid := strings.TrimSpace(f.inputs[radiusFieldServerPort].Value()) != ""
	secretValid := strings.TrimSpace(f.inputs[radiusFieldSecret].Value()) != ""
	canSave := serverValid && secretValid

	b.WriteString("\n")
	if canSave {
		b.WriteString(saveKeyStyle.Render(" [ Save ] ") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Enter", tr("保存 (Save)"), "Esc", tr("キャンセル")))
	} else {
		b.WriteString(dimStyle.Render("  [ Save - Please fill required fields ]") + "\n")
		b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "Esc", tr("キャンセル")))
	}
	return b.String()
}
