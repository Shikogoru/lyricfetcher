package i18n

import (
	"encoding/json"
	"os"
	"testing"
)

func TestInitAndTranslate(t *testing.T) {
	if err := Init("en"); err != nil {
		t.Fatalf("Init('en') failed: %v", err)
	}
	if CurrentLang() != "en" {
		t.Errorf("expected currentLang 'en', got %s", CurrentLang())
	}
	if T("file_header") != "[FILE]" {
		t.Errorf("expected '[FILE]', got %s", T("file_header"))
	}

	if err := Init("vi"); err != nil {
		t.Fatalf("Init('vi') failed: %v", err)
	}
	if CurrentLang() != "vi" {
		t.Errorf("expected currentLang 'vi', got %s", CurrentLang())
	}
	if T("file_header") != "[TỆP]" {
		t.Errorf("expected '[TỆP]', got %s", T("file_header"))
	}

	if err := Init("zh"); err != nil {
		t.Fatalf("Init('zh') failed: %v", err)
	}
	if CurrentLang() != "zh" {
		t.Errorf("expected currentLang 'zh', got %s", CurrentLang())
	}
	if T("file_header") != "[文件]" {
		t.Errorf("expected '[文件]', got %s", T("file_header"))
	}
}

func TestMissingKey(t *testing.T) {
	_ = Init("en")
	got := T("nonexistent")
	expected := "[MISSING:i18n]nonexistent"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestInvalidLanguageFallback(t *testing.T) {
	if err := Init("fr"); err != nil {
		t.Fatalf("Init('fr') should not return error, got %v", err)
	}
	if CurrentLang() != "en" {
		t.Errorf("expected fallback to 'en', got %s", CurrentLang())
	}
}

func TestEnvAutoDetection(t *testing.T) {
	// Test LYRICFETCHER_LANG
	os.Setenv("LYRICFETCHER_LANG", "zh")
	defer os.Unsetenv("LYRICFETCHER_LANG")

	if err := Init("auto"); err != nil {
		t.Fatalf("Init('auto') failed: %v", err)
	}
	if CurrentLang() != "zh" {
		t.Errorf("expected 'zh' from LYRICFETCHER_LANG, got %s", CurrentLang())
	}

	os.Unsetenv("LYRICFETCHER_LANG")

	// Test LANG
	os.Setenv("LANG", "vi_VN.UTF-8")
	defer os.Unsetenv("LANG")

	if err := Init("auto"); err != nil {
		t.Fatalf("Init('auto') failed: %v", err)
	}
	if CurrentLang() != "vi" {
		t.Errorf("expected 'vi' from LANG, got %s", CurrentLang())
	}
}

func TestLocalesParity(t *testing.T) {
	langs := []string{"en", "vi", "zh"}
	var keySets []map[string]bool

	for _, lang := range langs {
		data, err := localesFS.ReadFile("locales/" + lang + ".json")
		if err != nil {
			t.Fatalf("failed to read %s.json: %v", lang, err)
		}
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("failed to unmarshal %s.json: %v", lang, err)
		}
		keys := make(map[string]bool)
		for k := range m {
			keys[k] = true
		}
		keySets = append(keySets, keys)
	}

	enKeys := keySets[0]
	for i, lang := range langs[1:] {
		currKeys := keySets[i+1]
		for k := range enKeys {
			if !currKeys[k] {
				t.Errorf("locale %s is missing key %q", lang, k)
			}
		}
		for k := range currKeys {
			if !enKeys[k] {
				t.Errorf("locale %s has unexpected key %q not in en.json", lang, k)
			}
		}
	}
}
