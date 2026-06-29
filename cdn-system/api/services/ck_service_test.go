package services

import (
	"sync"
	"testing"
	"time"

	"cdn-api/models"
)

func TestFilterAccessLogLinesForInsertDropsStaleLines(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-30 * time.Minute).Format(time.RFC3339)
	stale := now.Add(-48 * time.Hour).Format(time.RFC3339)
	lines := []string{
		`{"time_iso8601":"` + fresh + `","host":"example.com","status":200}`,
		`{"time_iso8601":"` + stale + `","host":"example.com","status":200}`,
	}
	filtered := filterAccessLogLinesForInsert(lines)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 fresh line, got %d", len(filtered))
	}
	if !stringsContains(filtered[0], fresh) {
		t.Fatalf("expected fresh line kept, got %s", filtered[0])
	}
}

func TestFilterAccessLogLinesForInsertKeepsRecentBatch(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	lines := []string{
		`{"time_iso8601":"` + now + `","host":"cf.dtapi.cc","status":200}`,
	}
	filtered := filterAccessLogLinesForInsert(lines)
	if len(filtered) != 1 {
		t.Fatalf("expected recent line kept, got %d", len(filtered))
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCloneGlobalConfigForEdgeIsIndependent(t *testing.T) {
	src := &models.GlobalConfig{
		ErrorPages: map[string]models.ErrorPageDefinition{
			"403": {Template: "before"},
		},
	}
	cloned := cloneGlobalConfigForEdge(src)
	if cloned == nil || cloned == src {
		t.Fatalf("expected independent clone")
	}
	cloned.ErrorPages["403"] = models.ErrorPageDefinition{Template: "after"}
	if src.ErrorPages["403"].Template != "before" {
		t.Fatalf("clone mutation leaked into source")
	}
}

func TestHashConfigVersionConcurrentNoPanic(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cfg := &models.EdgeConfig{
				ErrorPages: map[string]models.ErrorPageDefinition{
					"403": {Template: "<html>" + string(rune('a'+n%26)) + "</html>"},
				},
				CCRules: map[int64][]models.EdgeCCRuleItem{
					int64(n%3 + 1): {{MatcherID: int64(n)}},
				},
			}
			_ = hashConfigVersion(cfg)
		}(i)
	}
	wg.Wait()
}
