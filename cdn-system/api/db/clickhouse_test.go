package db

import (
	"strings"
	"testing"
)

func TestClickHouseTableStmtsAccessSlowDiagnosisColumns(t *testing.T) {
	sql := strings.Join(clickHouseTableStmts(), "\n")
	for _, want := range []string{
		"upstream_connect_time Float64",
		"upstream_header_time Float64",
		"slow_reason String",
		"slow_advice String",
		"ADD COLUMN IF NOT EXISTS upstream_connect_time",
		"ADD COLUMN IF NOT EXISTS upstream_header_time",
		"ADD COLUMN IF NOT EXISTS slow_reason",
		"ADD COLUMN IF NOT EXISTS slow_advice",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("ClickHouse schema missing %q", want)
		}
	}
}
