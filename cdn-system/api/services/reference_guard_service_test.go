package services

import (
	"cdn-api/models"
	"testing"
)

func TestIsCCRuleInternal(t *testing.T) {
	if !IsCCRuleInternal(models.CCRule{Internal: true, UserID: 1}) {
		t.Fatalf("internal flag should protect rule")
	}
	if !IsCCRuleInternal(models.CCRule{UserID: 0}) {
		t.Fatalf("system rule should be protected")
	}
	if IsCCRuleInternal(models.CCRule{UserID: 9, Internal: false}) {
		t.Fatalf("user rule should not be protected")
	}
}

func TestGuardCCRuleTypeChange(t *testing.T) {
	if msg := GuardCCRuleTypeChange(true, "user"); msg != "cc_rule.system_type_locked" {
		t.Fatalf("msg = %q, want cc_rule.system_type_locked", msg)
	}
	if msg := GuardCCRuleTypeChange(true, "system"); msg != "" {
		t.Fatalf("system type should be allowed, got %q", msg)
	}
	if msg := GuardCCRuleTypeChange(false, "user"); msg != "" {
		t.Fatalf("user rule type change should be allowed, got %q", msg)
	}
}
