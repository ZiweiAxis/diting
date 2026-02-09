// Package audit 提供审计存储接口。
package audit

import (
	"context"

	"diting/internal/models"
)

// Store 审计存储接口：仅追加写；v1 支持按 trace_id 查询。
type Store interface {
	// Append 追加一条审计记录；Evidence 含 trace_id、policy_rule_id、decision_reason、CHEQ 状态、时间戳等。
	Append(ctx context.Context, e *models.Evidence) error
	// QueryByTraceID 按 trace_id 查询单请求完整决策链（MVP CLI/脚本用）。
	QueryByTraceID(ctx context.Context, traceID string) ([]*models.Evidence, error)
}
