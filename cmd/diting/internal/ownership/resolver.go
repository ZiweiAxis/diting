// Package ownership 提供资源/服务与确认人映射解析。
package ownership

import (
	"context"
)

// Resolver 根据资源与操作解析确认人标识列表，供 CHEQ 投递目标使用。
type Resolver interface {
	// Resolve 返回该 resource（及可选 action）对应的确认人 ID 列表（如飞书 user_id）。
	Resolve(ctx context.Context, resource, action string) (confirmerIDs []string, err error)
}
