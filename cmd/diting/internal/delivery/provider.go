package delivery

import (
	"context"
)

// Provider 投递接口：将待确认请求投递到 IM/CLI 等。
// MVP 实现：飞书、CLI；可被 cheq 包在 Create 后调用。
type Provider interface {
	// Deliver 投递待确认请求；Object 与 Options 由调用方组装。
	Deliver(ctx context.Context, in *DeliverInput) error
}
