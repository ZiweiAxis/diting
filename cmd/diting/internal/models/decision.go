package models

// DecisionKind 表示策略评估结果种类，与 AuthZEN 语义对齐。
type DecisionKind int

const (
	// DecisionAllow 允许放行。
	DecisionAllow DecisionKind = iota
	// DecisionDeny 拒绝，不转发。
	DecisionDeny
	// DecisionReview 需人工确认，进入 CHEQ。
	DecisionReview
)

// Decision 表示单次策略评估的决策结果。
type Decision struct {
	Kind             DecisionKind
	PolicyRuleID     string // 命中的策略规则 ID，审计可追溯。
	DecisionReason   string // 决策理由，满足可解释 v1。
}

// Allow 返回是否允许放行。
func (d *Decision) Allow() bool { return d.Kind == DecisionAllow }

// Deny 返回是否拒绝。
func (d *Decision) Deny() bool { return d.Kind == DecisionDeny }

// Review 返回是否需要人工确认。
func (d *Decision) Review() bool { return d.Kind == DecisionReview }
