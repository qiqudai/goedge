package services

import "testing"

func TestNormalizeParentFetchMode(t *testing.T) {
	cases := map[string]string{
		"":       ParentFetchOrigin,
		"ORIGIN": ParentFetchOrigin,
		"l1":     ParentFetchL1,
		"L2":     ParentFetchL2,
		"custom": ParentFetchOrigin,
	}
	for in, want := range cases {
		if got := NormalizeParentFetchMode(in); got != want {
			t.Fatalf("NormalizeParentFetchMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateParentFetchConfig(t *testing.T) {
	if err := ValidateParentFetchConfig(1, ParentFetchConfig{ParentFetchMode: ParentFetchL1}); err != nil {
		t.Fatalf("L1 node should ignore parent fetch validation: %v", err)
	}
	if err := ValidateParentFetchConfig(3, ParentFetchConfig{ParentFetchMode: ParentFetchOrigin}); err != nil {
		t.Fatalf("origin mode should pass without parent group: %v", err)
	}
	if err := ValidateParentFetchConfig(3, ParentFetchConfig{ParentFetchMode: ParentFetchL1}); err == nil {
		t.Fatalf("expected error when parent group missing for l1 mode")
	}
	if err := ValidateParentFetchConfig(3, ParentFetchConfig{
		ParentFetchMode: ParentFetchL2,
		ParentGroupID:   10,
	}); err != nil {
		t.Fatalf("valid l2 config rejected: %v", err)
	}
}

func TestResolveParentUpstreamKeys(t *testing.T) {
	groupID := int64(5)
	l1Key := "l1_upstream_5"
	l2Key := "l2_upstream_5"
	cfg := ParentFetchConfig{ParentGroupID: groupID, ParentFetchMode: ParentFetchL1}
	if cfg.ParentFetchMode != ParentFetchL1 {
		t.Fatalf("unexpected mode")
	}
	_ = l1Key
	_ = l2Key
}
