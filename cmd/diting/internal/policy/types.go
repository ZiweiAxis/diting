// Package policy 提供策略评估接口与 AuthZEN 风格类型。
package policy

// EvaluateRequest 为 PolicyEngine.Evaluate 的入参，与 AuthZEN Subject-Action-Resource-Context 一致。
// 可从 RequestContext 转换得到。
type EvaluateRequest struct {
	Subject  string            // Agent 或主体标识。
	Action   string            // 操作。
	Resource string            // 资源标识。
	Context  map[string]string  // 扩展上下文（可选）。
}
