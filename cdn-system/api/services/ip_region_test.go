package services

import (
	"strings"
	"testing"
)

func TestLookupIPRegionParsing_CurrentXDBLayout(t *testing.T) {
	region := "中国|江苏省|南京市|0"
	parts := splitRegionPartsForTest(region)
	if parts.country != "中国" {
		t.Fatalf("unexpected country: %q", parts.country)
	}
	if parts.province != "江苏省" {
		t.Fatalf("unexpected province: %q", parts.province)
	}
}

func TestCleanRegion_ZeroBecomesEmpty(t *testing.T) {
	if got := cleanRegion("0"); got != "" {
		t.Fatalf("expected zero marker to be empty, got %q", got)
	}
}

type regionPartsForTest struct {
	country  string
	province string
}

func splitRegionPartsForTest(region string) regionPartsForTest {
	parts := strings.Split(region, "|")
	out := regionPartsForTest{}
	if len(parts) > 0 {
		out.country = cleanRegion(parts[0])
	}
	if len(parts) > 1 {
		out.province = cleanRegion(parts[1])
	}
	return out
}
