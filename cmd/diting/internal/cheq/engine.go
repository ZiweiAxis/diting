package cheq

import (
	"context"

	"diting/internal/models"
)

// Engine CHEQ 统一确认引擎接口。
// Create 创建待确认对象；GetByID 查询状态；Submit 幂等提交确认结果。
// 超时语义：由实现侧在 GetByID 或后台任务中将过期对象置为 expired。
type Engine interface {
	// Create 创建 ConfirmationObject，返回带 ID 的对象；后续可投递并等待 Submit。
	Create(ctx context.Context, in *CreateInput) (*models.ConfirmationObject, error)
	// GetByID 根据 id 查询当前状态；若已过期则 Status 为 expired。
	GetByID(ctx context.Context, id string) (*models.ConfirmationObject, error)
	// Submit 幂等提交确认结果；已处理或已过期返回 ErrAlreadyProcessed/ErrExpired。
	// confirmerID 用于 I-008「全部通过」时记录谁批准；为空时按「任一通过」处理。
	Submit(ctx context.Context, id string, approved bool, confirmerID string) error
}
