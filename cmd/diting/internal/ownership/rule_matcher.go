// Package ownership 提供 I-009 按 path/risk_level 匹配审批规则（超时、审批人、策略）。
package ownership

import (
	"strings"
)

// ApprovalRuleMatch 单条规则匹配结果：超时秒数、审批人 ID 列表、审批策略（any/all）。
type ApprovalRuleMatch struct {
	TimeoutSeconds  int
	ApprovalUserIDs []string
	ApprovalPolicy  string // "any" 或 "all"
}

// RuleMatcher I-009：按 path 前缀与 risk_level 匹配审批规则，返回超时与审批人；无匹配时返回默认值。
type RuleMatcher struct {
	rules  []ruleEntry
	def    ApprovalRuleMatch
}

type ruleEntry struct {
	pathPrefix string
	riskLevel  string
	timeoutSec int
	userIDs    []string
	policy     string
}

// NewRuleMatcher 从规则列表与默认值构建匹配器。rules 为 (path_prefix, risk_level, timeout_seconds, approval_user_ids, approval_policy)。
// defaultMatch 为无匹配时使用的超时、审批人、策略。
func NewRuleMatcher(rules []struct {
	PathPrefix      string
	RiskLevel       string
	TimeoutSeconds  int
	ApprovalUserIDs []string
	ApprovalPolicy  string
}, defaultMatch ApprovalRuleMatch) *RuleMatcher {
	entries := make([]ruleEntry, 0, len(rules))
	for _, r := range rules {
		policy := r.ApprovalPolicy
		if policy != "all" {
			policy = "any"
		}
		entries = append(entries, ruleEntry{
			pathPrefix: r.PathPrefix,
			riskLevel:  r.RiskLevel,
			timeoutSec: r.TimeoutSeconds,
			userIDs:    append([]string(nil), r.ApprovalUserIDs...),
			policy:     policy,
		})
	}
	if defaultMatch.ApprovalPolicy != "all" {
		defaultMatch.ApprovalPolicy = "any"
	}
	return &RuleMatcher{rules: entries, def: defaultMatch}
}

// Match 按 path 与 riskLevel 匹配第一条规则；path 为资源路径（如 /api/delete），riskLevel 可为空。
// 返回该规则的超时、审批人、策略；无匹配时返回默认值。
func (m *RuleMatcher) Match(path, riskLevel string) ApprovalRuleMatch {
	for _, e := range m.rules {
		if e.pathPrefix != "" && !strings.HasPrefix(path, e.pathPrefix) {
			continue
		}
		if e.riskLevel != "" && e.riskLevel != riskLevel {
			continue
		}
		out := ApprovalRuleMatch{
			TimeoutSeconds:  e.timeoutSec,
			ApprovalUserIDs: append([]string(nil), e.userIDs...),
			ApprovalPolicy:  e.policy,
		}
		if out.TimeoutSeconds <= 0 {
			out.TimeoutSeconds = m.def.TimeoutSeconds
		}
		if len(out.ApprovalUserIDs) == 0 {
			out.ApprovalUserIDs = append([]string(nil), m.def.ApprovalUserIDs...)
		}
		return out
	}
	return m.def
}
