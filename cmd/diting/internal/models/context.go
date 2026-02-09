// Package models 提供 proxy、policy、cheq、audit 等组件共用的请求上下文与数据类型。
package models

import "net/http"

// RequestContext 表示单次请求的上下文，供 L0/L1/L2 使用。
// 含 Agent 身份、目标 URL/方法、资源标识、操作、请求头等。
type RequestContext struct {
	// AgentIdentity L0 身份标识（如 API Key、user_id）；空表示未识别。
	AgentIdentity string
	// Method HTTP 方法（GET、POST、CONNECT 等）。
	Method string
	// TargetURL 目标 URL（代理转发时的上游地址或路径）。
	TargetURL string
	// Resource 资源标识，用于 L2 策略评估（AuthZEN Resource）。
	Resource string
	// Action 操作标识，用于 L2 策略评估（AuthZEN Action）。
	Action string
	// Headers 请求头副本，可含 traceparent、X-Agent-Token 等。
	Headers http.Header
	// 可扩展：Subject、Context 等 AuthZEN 字段可从 Headers/AgentIdentity 解析后填入。
}
