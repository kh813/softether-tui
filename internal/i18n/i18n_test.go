package i18n

import "testing"

func setLocaleEnv(t *testing.T, lcAll, lcMessages, lang string) {
	t.Helper()
	t.Setenv("LC_ALL", lcAll)
	t.Setenv("LC_MESSAGES", lcMessages)
	t.Setenv("LANG", lang)
}

func TestDetectDefaultsToEnglishWhenUnset(t *testing.T) {
	setLocaleEnv(t, "", "", "")
	if got := Detect(); got != EN {
		t.Errorf("Detect() = %q, want %q", got, EN)
	}
}

func TestDetectJapaneseFromLang(t *testing.T) {
	setLocaleEnv(t, "", "", "ja_JP.UTF-8")
	if got := Detect(); got != JA {
		t.Errorf("Detect() = %q, want %q", got, JA)
	}
}

func TestDetectEnglishFromLang(t *testing.T) {
	setLocaleEnv(t, "", "", "en_US.UTF-8")
	if got := Detect(); got != EN {
		t.Errorf("Detect() = %q, want %q", got, EN)
	}
}

func TestDetectNonUTF8OrCLocaleDefaultsToEnglish(t *testing.T) {
	setLocaleEnv(t, "", "", "C")
	if got := Detect(); got != EN {
		t.Errorf("Detect() = %q, want %q", got, EN)
	}
}

func TestDetectPrecedenceLCAllOverridesLang(t *testing.T) {
	// LC_ALL says English even though LANG says Japanese: LC_ALL must win.
	setLocaleEnv(t, "en_US.UTF-8", "", "ja_JP.UTF-8")
	if got := Detect(); got != EN {
		t.Errorf("Detect() = %q, want %q (LC_ALL should override LANG)", got, EN)
	}
}

func TestDetectPrecedenceLCMessagesOverridesLang(t *testing.T) {
	setLocaleEnv(t, "", "ja_JP.UTF-8", "en_US.UTF-8")
	if got := Detect(); got != JA {
		t.Errorf("Detect() = %q, want %q (LC_MESSAGES should override LANG)", got, JA)
	}
}

func TestParse(t *testing.T) {
	cases := map[string]Lang{
		"en": EN, "EN": EN, " en ": EN, "english": EN,
		"ja": JA, "JA": JA, "japanese": JA, "jp": JA,
	}
	for input, want := range cases {
		got, err := Parse(input)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("Parse(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("fr"); err == nil {
		t.Error("Parse(\"fr\") should have returned an error")
	}
}
