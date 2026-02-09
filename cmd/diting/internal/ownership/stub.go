package ownership

import (
	"context"
)

// StubResolver 占位实现：恒返回空列表，供 Phase 2 装配。
type StubResolver struct{}

func (StubResolver) Resolve(ctx context.Context, resource, action string) ([]string, error) {
	return nil, nil
}
