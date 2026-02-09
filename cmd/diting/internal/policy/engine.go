package policy

import (
	"context"

	"diting/internal/models"
)

// Engine 策略引擎接口（AuthZEN PDP 语义）。
// 仅接口定义，无实现；实现见 Phase 4 内置或 OPA 对接。
type Engine interface {
	// Evaluate 根据请求上下文做 L2 策略评估，返回 Allow/Deny/Review 及 policy_rule_id、decision_reason。
	Evaluate(ctx context.Context, req *models.RequestContext) (*models.Decision, error)
}
