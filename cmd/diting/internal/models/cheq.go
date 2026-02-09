package models

import "time"

// ConfirmationStatus 表示 ConfirmationObject 的生命周期状态。
type ConfirmationStatus string

const (
	ConfirmationStatusPending   ConfirmationStatus = "pending"
	ConfirmationStatusDelivered ConfirmationStatus = "delivered"
	ConfirmationStatusApproved ConfirmationStatus = "approved"
	ConfirmationStatusRejected ConfirmationStatus = "rejected"
	ConfirmationStatusExpired  ConfirmationStatus = "expired"
)

// ConfirmationObject 表示一次待确认对象（CHEQ 统一确认协议）。
// 含 id、trace_id、状态、创建/过期时间、关联资源与操作、确认人标识等。
type ConfirmationObject struct {
	ID         string
	TraceID    string
	Status     ConfirmationStatus
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Resource   string
	Action     string
	Summary    string             // 操作摘要，供确认人查看。
	ConfirmerIDs []string         // 确认人标识列表（如飞书 user_id）。
	Type       string             // 如 agent_onboarding / service_access / operation_approval。
}

// IsTerminal 返回是否已终态（不再接受 Submit）。
func (c *ConfirmationObject) IsTerminal() bool {
	return c.Status == ConfirmationStatusApproved ||
		c.Status == ConfirmationStatusRejected ||
		c.Status == ConfirmationStatusExpired
}
