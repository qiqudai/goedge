package services

import (
	"strings"
	"testing"
	"time"
)

func TestInterpolateQuery_IgnoresQuestionMarksInsideQuotes(t *testing.T) {
	query := "SELECT extract(http_referer, '^(?:https?://)?([^/?#]+)') AS item FROM node_access_logs WHERE ts >= ? AND ts <= ? AND item LIKE ?"
	start := time.Unix(100, 0).UTC()
	end := time.Unix(200, 0).UTC()

	got := interpolateQuery(query, start, end, "%ali%")

	if strings.Contains(got, "httpstoDateTime") {
		t.Fatalf("regex question mark should not be replaced: %s", got)
	}
	if strings.Count(got, "toDateTime(") != 2 {
		t.Fatalf("expected two time placeholders to be replaced, got: %s", got)
	}
	if !strings.Contains(got, "'%ali%'") {
		t.Fatalf("expected keyword placeholder replacement, got: %s", got)
	}
}
