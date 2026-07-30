// Package i18n selects the softether-tui UI language: which locale
// environment variables to trust, in what order, and what the safe default
// is when none of them indicate a preference.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a UI language softether-tui can render in.
type Lang string

const (
	// EN is the default: plain ASCII text that renders correctly
	// regardless of the terminal's locale/encoding support.
	EN Lang = "en"
	JA Lang = "ja"
)

// Detect selects a language from the environment, following POSIX locale
// precedence: LC_ALL overrides LC_MESSAGES overrides LANG. The first of
// those that is set decides the outcome; if none are set, it defaults to
// EN. This exists because softether-tui's Japanese UI text was found to
// render as mojibake on servers without a Japanese (UTF-8) locale — English
// is the safe fallback since it doesn't depend on multi-byte rendering.
func Detect() Lang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "ja") {
			return JA
		}
		// A non-Japanese value at the highest set precedence level wins
		// outright; POSIX semantics don't fall through past a set variable.
		return EN
	}
	return EN
}

// Parse validates a --lang flag value.
func Parse(s string) (Lang, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "en", "english":
		return EN, nil
	case "ja", "japanese", "jp":
		return JA, nil
	default:
		return "", fmt.Errorf("unsupported language %q (must be \"en\" or \"ja\")", s)
	}
}
