package i18n

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var (
	loadOnce sync.Once
	loaded   bool
	messages map[string]string
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
