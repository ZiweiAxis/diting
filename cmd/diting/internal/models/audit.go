package models

import "time"

// Evidence 表示一条审计记录（OTel 规范、可配置脱敏）。
// 字段名与架构约定一致，便于 JSON 序列化为 snake_case。
type Evidence struct {
	TraceID         string    `json:"trace_id"`
	SpanID          string    `json:"span_id,omitempty"`
	AgentID         string    `json:"agent_id,omitempty"`
	PolicyRuleID    string    `json:"policy_rule_id,omitempty"`
	DecisionReason  string    `json:"decision_reason,omitempty"`
	Decision        string    `json:"decision"` // allow / deny / review / approved / rejected / expired
	CHEQStatus      string    `json:"cheq_status,omitempty"`
	Confirmer       string    `json:"confirmer,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	Resource        string    `json:"resource,omitempty"`
	Action          string    `json:"action,omitempty"`
	// 可扩展：L0/L1/L2 各层 decision、request_id 等。
}
