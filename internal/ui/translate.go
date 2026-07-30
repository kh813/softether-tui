package ui

import "softether-tui/internal/i18n"

// lang defaults to EN: a safe, ASCII-only fallback if SetLang is never
// called (e.g. in tests that construct screens directly).
var lang = i18n.EN

// SetLang selects which language tr() translates into. Call this once from
// main(), before New(), based on --lang or i18n.Detect(). softether-tui is
// single-threaded (one Bubble Tea event loop), so a package-level var set
// once at startup is safe here.
func SetLang(l i18n.Lang) {
	lang = l
}

// t translates a Japanese UI string into English via enCatalog when lang is
// i18n.EN. Every literal in this package is written in Japanese and passed
// through tr(); an entry missing from enCatalog is a bug (falls back to the
// Japanese source, which is safer than a runtime panic but still worth
// fixing - see TestEnCatalogCoversAllSourceStrings).
func tr(ja string) string {
	if lang != i18n.EN {
		return ja
	}
	if en, ok := enCatalog[ja]; ok {
		return en
	}
	return ja
}
