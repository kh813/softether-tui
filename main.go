// Command softether-tui is a TUI wrapper around SoftEther's vpncmd.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"softether-tui/internal/config"
	"softether-tui/internal/i18n"
	"softether-tui/internal/ui"
	"softether-tui/internal/vpncmd"
)

// version, commit and date are overridden at build time via -ldflags (see
// app_specs.md 11.1 and Makefile's LDFLAGS).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("softether-tui %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	langFlag := flag.String("lang", "", `UI language: "en" or "ja" (default: auto-detect from LC_ALL/LC_MESSAGES/LANG, falling back to en)`)
	flag.Parse()

	lang := i18n.Detect()
	if *langFlag != "" {
		l, err := i18n.Parse(*langFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		lang = l
	}
	ui.SetLang(lang)

	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18nErr(lang, "設定ディレクトリの取得に失敗しました:", "failed to resolve config directory:"), err)
		os.Exit(1)
	}
	store := config.NewStore(path)

	binaryPath, err := vpncmd.Locate()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18nErr(lang, "警告: vpncmd が PATH 上に見つかりません。接続系の操作は失敗します。", "warning: vpncmd was not found in PATH; connection-related operations will fail."))
	}
	client := vpncmd.NewClient(binaryPath)

	model := ui.New(store, client, version)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, i18nErr(lang, "エラー:", "error:"), err)
		os.Exit(1)
	}
}

// i18nErr picks between the Japanese and English variant of a startup
// message. These few strings live in main.go rather than internal/ui's
// catalog because they can be printed before ui.New/SetLang have run.
func i18nErr(lang i18n.Lang, ja, en string) string {
	if lang == i18n.JA {
		return ja
	}
	return en
}
