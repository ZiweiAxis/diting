package delivery

import (
	"context"
)

// StubProvider 占位实现：Deliver 无操作，供 Phase 2 装配。
type StubProvider struct{}

func (StubProvider) Deliver(ctx context.Context, in *DeliverInput) error {
	return nil
}
