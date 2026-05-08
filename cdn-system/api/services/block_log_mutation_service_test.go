package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExecClickHouseMutationHTTP_UsesPOST(t *testing.T) {
	var gotMethod string
	var gotQuery string
	var gotDB string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotDB = r.URL.Query().Get("database")
		body, _ := io.ReadAll(r.Body)
		gotQuery = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &httpCKConfig{
		baseURL:  ts.URL,
		user:     "default",
		pass:     "",
		database: "cdn_logs",
	}
	query := "ALTER TABLE node_access_logs DELETE WHERE remote_addr = '1.1.1.1'"
	if err := execClickHouseMutationHTTP(cfg, query); err != nil {
		t.Fatalf("execClickHouseMutationHTTP failed: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotDB != "cdn_logs" {
		t.Fatalf("expected database=cdn_logs, got %s", gotDB)
	}
	if strings.TrimSpace(gotQuery) != query {
		t.Fatalf("unexpected query body: %s", gotQuery)
	}
}

