// Package cheq 提供 CHEQ 确认引擎接口与类型。
package cheq

import (
	"errors"
	"time"
)

// CreateInput 创建 ConfirmationObject 的入参。
type CreateInput struct {
	TraceID       string
	Resource      string
	Action        string
	Summary       string
	ExpiresAt     time.Time
	ConfirmerIDs  []string
	Type          string
	ApprovalPolicy string // I-009：本请求的审批策略（any/all）；空则用引擎默认
}

// ErrAlreadyProcessed 表示该 ConfirmationObject 已处理（幂等提交时返回）。
var ErrAlreadyProcessed = errors.New("cheq: confirmation object already processed")

// ErrNotFound 表示 ConfirmationObject 不存在。
var ErrNotFound = errors.New("cheq: confirmation object not found")

// ErrExpired 表示已过期。
var ErrExpired = errors.New("cheq: confirmation object expired")
