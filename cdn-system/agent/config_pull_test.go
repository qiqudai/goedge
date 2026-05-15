package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPullConfigSendsLocalVersionAndAcceptsNotModified(t *testing.T) {
	root := t.TempDir()
	prevAPI := API_BaseURL
	prevNodeID := NodeID
	prevToken := AuthToken
	prevConfigPath := CONFIG_PATH
	prevConfigBak := CONFIG_BAK
	defer func() {
		API_BaseURL = prevAPI
		NodeID = prevNodeID
		AuthToken = prevToken
		CONFIG_PATH = prevConfigPath
		CONFIG_BAK = prevConfigBak
	}()

	CONFIG_PATH = filepath.Join(root, "cdn_config.json")
	CONFIG_BAK = filepath.Join(root, "cdn_config.json.bak")
	if err := os.WriteFile(CONFIG_PATH, []byte(`{"version":123}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	NodeID = "41"
	AuthToken = "token"

	var gotVersion string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVersion = r.URL.Query().Get("version")
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Query().Get("node_id") != "41" {
			t.Fatalf("node_id = %q, want 41", r.URL.Query().Get("node_id"))
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()
	API_BaseURL = server.URL

	if err := pullConfig(); err != nil {
		t.Fatalf("pullConfig returned error: %v", err)
	}
	if gotVersion != "123" {
		t.Fatalf("version query = %q, want 123", gotVersion)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("Authorization = %q, want Bearer token", gotAuth)
	}
}

func TestPullConfigTreatsEmptyOKAsNoChange(t *testing.T) {
	root := t.TempDir()
	prevAPI := API_BaseURL
	prevNodeID := NodeID
	prevToken := AuthToken
	prevConfigPath := CONFIG_PATH
	prevConfigBak := CONFIG_BAK
	defer func() {
		API_BaseURL = prevAPI
		NodeID = prevNodeID
		AuthToken = prevToken
		CONFIG_PATH = prevConfigPath
		CONFIG_BAK = prevConfigBak
	}()

	CONFIG_PATH = filepath.Join(root, "cdn_config.json")
	CONFIG_BAK = filepath.Join(root, "cdn_config.json.bak")
	if err := os.WriteFile(CONFIG_PATH, []byte(`{"version":123}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	NodeID = "41"
	AuthToken = "token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	API_BaseURL = server.URL

	if err := pullConfig(); err != nil {
		t.Fatalf("pullConfig returned error: %v", err)
	}
}

func TestConfigPullInitialDelaySpreadsNumericNodeIDs(t *testing.T) {
	prevNodeID := NodeID
	defer func() {
		NodeID = prevNodeID
	}()

	NodeID = "41"
	got := configPullInitialDelay(60 * time.Second)
	want := 16 * time.Second
	if got != want {
		t.Fatalf("initial delay = %s, want %s", got, want)
	}
}
