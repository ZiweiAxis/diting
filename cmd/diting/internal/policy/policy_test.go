package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"diting/internal/models"
)

func TestLoadRules_EmptyPath(t *testing.T) {
	rules, err := LoadRules("")
	if err != nil {
		t.Fatalf("LoadRules(\"\"): %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestLoadRules_MissingFile(t *testing.T) {
	rules, err := LoadRules(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("LoadRules(missing): %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for missing file, got %d", len(rules))
	}
}

func TestLoadRules_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	content := []byte(`
rules:
  - id: r1
    subject: "agent-a"
    action: "read"
    resource: "/api/data"
    decision: allow
  - id: r2
    subject: "*"
    action: "*"
    resource: "*"
    decision: deny
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadRules(path)
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].ID != "r1" || rules[0].Decision != RuleAllow {
		t.Errorf("first rule: id=%q decision=%q", rules[0].ID, rules[0].Decision)
	}
	if rules[1].ID != "r2" || rules[1].Decision != RuleDeny {
		t.Errorf("second rule: id=%q decision=%q", rules[1].ID, rules[1].Decision)
	}
}

func TestRule_Match(t *testing.T) {
	tests := []struct {
		name     string
		r        Rule
		sub, act, res string
		want     bool
	}{
		{"exact", Rule{Subject: "a", Action: "b", Resource: "c"}, "a", "b", "c", true},
		{"subject mismatch", Rule{Subject: "a", Action: "b", Resource: "c"}, "x", "b", "c", false},
		{"wildcard subject", Rule{Subject: "*", Action: "b", Resource: "c"}, "any", "b", "c", true},
		{"empty subject", Rule{Subject: "", Action: "b", Resource: "c"}, "any", "b", "c", true},
		{"all wildcard", Rule{Subject: "*", Action: "*", Resource: "*"}, "x", "y", "z", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Match(tt.sub, tt.act, tt.res)
			if got != tt.want {
				t.Errorf("Match(%q,%q,%q) = %v, want %v", tt.sub, tt.act, tt.res, got, tt.want)
			}
		})
	}
}

func TestEngineImpl_Evaluate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	content := []byte(`
rules:
  - id: allow-read
    subject: "agent-1"
    action: "GET"
    resource: "/api/read"
    decision: allow
  - id: deny-all
    subject: "*"
    action: "*"
    resource: "*"
    decision: deny
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngineImpl(path)
	if err != nil {
		t.Fatalf("NewEngineImpl: %v", err)
	}
	ctx := context.Background()

	// Allow: agent-1 GET /api/read
	req1 := &models.RequestContext{AgentIdentity: "agent-1", Method: "GET", TargetURL: "/api/read"}
	dec1, err := eng.Evaluate(ctx, req1)
	if err != nil {
		t.Fatalf("Evaluate allow: %v", err)
	}
	if dec1.Kind != models.DecisionAllow {
		t.Errorf("expected Allow, got %v", dec1.Kind)
	}
	if dec1.PolicyRuleID != "allow-read" {
		t.Errorf("expected rule allow-read, got %q", dec1.PolicyRuleID)
	}

	// Deny: other agent
	req2 := &models.RequestContext{AgentIdentity: "agent-2", Method: "GET", TargetURL: "/api/read"}
	dec2, err := eng.Evaluate(ctx, req2)
	if err != nil {
		t.Fatalf("Evaluate deny: %v", err)
	}
	if dec2.Kind != models.DecisionDeny {
		t.Errorf("expected Deny, got %v", dec2.Kind)
	}
}
