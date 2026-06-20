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

func TestGuardCCRuleGroupDeleteSystemProtected(t *testing.T) {
	rule := models.CCRule{ID: 1, Internal: true, Enable: true, UserID: 0}
	msg, err := GuardCCRuleGroupDelete(rule)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if msg != "cc_rule.system_protected" {
		t.Fatalf("msg = %q, want cc_rule.system_protected", msg)
	}
}
