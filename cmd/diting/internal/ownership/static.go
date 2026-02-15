package ownership

import (
	"context"
	"sync"
)

// StaticResolver 从静态映射解析确认人：resource 或 "*" 对应 confirmer ID 列表；无匹配时返回 defaultIDs（I-008）。
type StaticResolver struct {
	mu         sync.RWMutex
	m          map[string][]string // resource -> confirmer_ids
	defaultIDs []string            // 无匹配时返回（如 config approval_user_ids）
}

// NewStaticResolver 根据 resource -> confirmer_ids 映射创建；defaultIDs 为无匹配时的默认审批人列表，可为 nil。
func NewStaticResolver(staticMap map[string][]string, defaultIDs []string) *StaticResolver {
	if staticMap == nil {
		staticMap = make(map[string][]string)
	}
	m := make(map[string][]string, len(staticMap))
	for k, v := range staticMap {
		ids := make([]string, len(v))
		copy(ids, v)
		m[k] = ids
	}
	var def []string
	if len(defaultIDs) > 0 {
		def = make([]string, len(defaultIDs))
		copy(def, defaultIDs)
	}
	return &StaticResolver{m: m, defaultIDs: def}
}

// Resolve 先查 resource，再查 "*"；无则返回 defaultIDs（可能为空）。
func (s *StaticResolver) Resolve(ctx context.Context, resource, action string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ids, ok := s.m[string(resource)]; ok && len(ids) > 0 {
		return append([]string(nil), ids...), nil
	}
	if ids, ok := s.m["*"]; ok && len(ids) > 0 {
		return append([]string(nil), ids...), nil
	}
	if len(s.defaultIDs) > 0 {
		return append([]string(nil), s.defaultIDs...), nil
	}
	return nil, nil
}
