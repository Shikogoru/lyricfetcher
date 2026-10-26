package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	mu          sync.RWMutex
	currentLang string
	messages    map[string]string
)

var supportedLangs = map[string]bool{
	"en": true,
	"vi": true,
	"zh": true,
}

// detectLang detects language from environment variables.
func detectLang() string {
	if val := os.Getenv("LYRICFETCHER_LANG"); val != "" {
		return val
	}
	if val := os.Getenv("LANG"); val != "" {
		return val
	}
	if val := os.Getenv("LANGUAGE"); val != "" {
		return val
	}
	return "en"
}

// normalizeLang standardizes language strings (e.g., "zh-CN", "vi_VN.UTF-8") to basic codes.
func normalizeLang(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.ReplaceAll(s, "-", "_")
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "_"); idx != -1 {
		s = s[:idx]
	}
	return s
}

// Init initializes the i18n system with the requested language code.
// If lang is empty or "auto", it auto-detects from environment variables.
// If the language is invalid, it prints a warning to stderr and falls back to "en".
func Init(lang string) error {
	trimmed := strings.TrimSpace(lang)
	if trimmed == "" || strings.ToLower(trimmed) == "auto" {
		trimmed = detectLang()
	}

	norm := normalizeLang(trimmed)
	if !supportedLangs[norm] {
		fmt.Fprintf(os.Stderr, "[WARN] invalid language %q, falling back to English\n", lang)
		norm = "en"
	}

	return loadLocale(norm)
}

func loadLocale(lang string) error {
	filePath := "locales/" + lang + ".json"
	data, err := localesFS.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded locale file %s: %w", filePath, err)
	}

	var msgs map[string]string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return fmt.Errorf("failed to unmarshal locale file %s: %w", filePath, err)
	}

	mu.Lock()
	defer mu.Unlock()
	currentLang = lang
	messages = msgs
	return nil
}

// SetLang switches the active language at runtime.
func SetLang(lang string) error {
	return Init(lang)
}

// CurrentLang returns the currently active language code.
func CurrentLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T translates a message key to the current language.
// If the key is not found, it returns "[MISSING:i18n]" + key.
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if messages != nil {
		if val, ok := messages[key]; ok {
			return val
		}
	}
	return "[MISSING:i18n]" + key
}

// Tf formats a translated string using fmt.Sprintf.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
