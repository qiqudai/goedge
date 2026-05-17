package services

import (
	"strings"
	"testing"
)

func TestBlockedStatusConditionExcludesOriginResponses(t *testing.T) {
	condition := blockedStatusCondition()
	if !strings.Contains(condition, "status IN (403,418,429,451,410)") {
		t.Fatalf("blocked status condition missing blocked codes: %s", condition)
	}
	if !strings.Contains(condition, "block_source != 'origin'") {
		t.Fatalf("blocked status condition must exclude explicit origin responses: %s", condition)
	}
	if !strings.Contains(condition, "NOT (block_source = '' AND upstream_addr != '')") {
		t.Fatalf("blocked status condition must exclude legacy inferred origin responses: %s", condition)
	}
}
