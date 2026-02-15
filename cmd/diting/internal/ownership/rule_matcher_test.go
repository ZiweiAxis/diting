package ownership

import (
	"testing"
)

func TestRuleMatcher_Match(t *testing.T) {
	def := ApprovalRuleMatch{
		TimeoutSeconds:  120,
		ApprovalUserIDs: []string{"default"},
		ApprovalPolicy:  "any",
	}
	rules := []struct {
		PathPrefix      string
		RiskLevel       string
		TimeoutSeconds  int
		ApprovalUserIDs []string
		ApprovalPolicy  string
	}{
		{PathPrefix: "/admin", RiskLevel: "high", TimeoutSeconds: 600, ApprovalUserIDs: []string{"a1", "a2"}, ApprovalPolicy: "all"},
		{PathPrefix: "/api", TimeoutSeconds: 60},
	}
	m := NewRuleMatcher(rules, def)

	// 无匹配 -> 默认
	got := m.Match("/other", "")
	if got.TimeoutSeconds != 120 || len(got.ApprovalUserIDs) != 1 || got.ApprovalUserIDs[0] != "default" || got.ApprovalPolicy != "any" {
		t.Errorf("Match(/other): got timeout=%d ids=%v policy=%s", got.TimeoutSeconds, got.ApprovalUserIDs, got.ApprovalPolicy)
	}

	// path 匹配 /admin，risk_level 匹配 high
	got = m.Match("/admin/delete", "high")
	if got.TimeoutSeconds != 600 || len(got.ApprovalUserIDs) != 2 || got.ApprovalPolicy != "all" {
		t.Errorf("Match(/admin/delete, high): got timeout=%d ids=%v policy=%s", got.TimeoutSeconds, got.ApprovalUserIDs, got.ApprovalPolicy)
	}

	// path 匹配 /admin 但 risk_level 不匹配 -> 不匹配此条，无下条 path 匹配 -> 默认
	got = m.Match("/admin/delete", "low")
	if got.TimeoutSeconds != 120 {
		t.Errorf("Match(/admin/delete, low): want default timeout 120, got %d", got.TimeoutSeconds)
	}

	// path 匹配 /api，无 risk_level -> 匹配第二条，timeout 60，审批人用默认
	got = m.Match("/api/read", "")
	if got.TimeoutSeconds != 60 || len(got.ApprovalUserIDs) != 1 || got.ApprovalUserIDs[0] != "default" {
		t.Errorf("Match(/api/read): got timeout=%d ids=%v", got.TimeoutSeconds, got.ApprovalUserIDs)
	}
}
