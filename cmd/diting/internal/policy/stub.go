package policy

import (
	"context"

	"diting/internal/models"
)

// StubEngine 占位实现：恒返回 Allow，供 Phase 2 装配与测试。
type StubEngine struct{}

func (StubEngine) Evaluate(ctx context.Context, req *models.RequestContext) (*models.Decision, error) {
	return &models.Decision{
		Kind:           models.DecisionAllow,
		PolicyRuleID:   "stub",
		DecisionReason: "stub allow",
	}, nil
}
