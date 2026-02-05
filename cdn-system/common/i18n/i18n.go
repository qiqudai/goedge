package i18n

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	loadOnce sync.Once
	loaded   bool
	messages map[string]string
	reverse  map[string]string
	loadErr  error
)

//go:embed messages.json
var embeddedMessages []byte

// Load reads the messages JSON into a read-only map.
func Load(path string) error {
	loadOnce.Do(func() {
		candidates := resolvePaths(path)
		var data []byte
		for _, p := range candidates {
			data, loadErr = os.ReadFile(p)
			if loadErr == nil {
				break
			}
		}
		if loadErr != nil {
			if len(embeddedMessages) == 0 {
				return
			}
			data = embeddedMessages
			loadErr = nil
		}
		var parsed map[string]string
		if err := json.Unmarshal(data, &parsed); err != nil {
			loadErr = err
			return
		}
		messages = parsed
		reverse = make(map[string]string, len(messages))
		for key, val := range messages {
			if val == "" {
				continue
			}
			if _, exists := reverse[val]; !exists {
				reverse[val] = key
			}
		}
		loaded = true
	})
	return loadErr
}

// T returns the localized message for key.
func T(key string) string {
	if !loaded {
		return key
	}
	if val, ok := messages[key]; ok {
		return val
	}
	return key
}

// NormalizeLang returns a normalized language code (zh/en) or empty when unknown.
func NormalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return ""
	}
	if strings.HasPrefix(lang, "zh") || lang == "cn" {
		return "zh"
	}
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	return ""
}

// Translate returns a localized message based on requested language.
// If lang is empty or unsupported, it falls back to zh when available.
func Translate(lang, message string) string {
	if message == "" {
		return message
	}
	lang = NormalizeLang(lang)
	if lang == "" {
		lang = "zh"
	}
	switch lang {
	case "en":
		// If message is a key, return key itself (English).
		if _, ok := messages[message]; ok {
			return message
		}
		// If message is a localized value, map back to key.
		if key, ok := reverse[message]; ok {
			return key
		}
		return message
	default: // zh
		if val, ok := messages[message]; ok {
			return val
		}
		return message
	}
}

func resolvePaths(path string) []string {
	paths := make([]string, 0, 6)
	if path != "" {
		paths = append(paths, path)
	}
	if env := os.Getenv("I18N_PATH"); env != "" {
		paths = append(paths, env)
	}
	paths = append(paths, filepath.Join("common", "i18n", "messages.json"))
	paths = append(paths, filepath.Join("..", "common", "i18n", "messages.json"))
	paths = append(paths, filepath.Join("i18n", "messages.json"))

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths = append(paths, filepath.Join(exeDir, "common", "i18n", "messages.json"))
		paths = append(paths, filepath.Join(exeDir, "..", "common", "i18n", "messages.json"))
		paths = append(paths, filepath.Join(exeDir, "i18n", "messages.json"))
	}

	unique := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := unique[p]; ok {
			continue
		}
		unique[p] = struct{}{}
		result = append(result, p)
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

// MustLoad ensures messages are loaded or returns an error.
func MustLoad(path string) error {
	if err := Load(path); err != nil {
		return err
	}
	if !loaded {
		return errors.New("i18n messages not loaded")
	}
	return nil
}
