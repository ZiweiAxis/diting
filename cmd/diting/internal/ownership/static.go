package ownership

import (
	"context"
	"sync"
)

// StaticResolver 从静态映射解析确认人：resource 或 "*" 对应 confirmer ID 列表。
type StaticResolver struct {
	mu   sync.RWMutex
	m    map[string][]string // resource -> confirmer_ids
}

// NewStaticResolver 根据 resource -> confirmer_ids 映射创建；传 nil 表示无映射。
func NewStaticResolver(staticMap map[string][]string) *StaticResolver {
	if staticMap == nil {
		staticMap = make(map[string][]string)
	}
	// 深拷贝，避免外部修改
	m := make(map[string][]string, len(staticMap))
	for k, v := range staticMap {
		ids := make([]string, len(v))
		copy(ids, v)
		m[k] = ids
	}
	return &StaticResolver{m: m}
}

// Resolve 先查 resource，再查 "*"；无则返回空列表。
func (s *StaticResolver) Resolve(ctx context.Context, resource, action string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ids, ok := s.m[string(resource)]; ok && len(ids) > 0 {
		return append([]string(nil), ids...), nil
	}
	if ids, ok := s.m["*"]; ok && len(ids) > 0 {
		return append([]string(nil), ids...), nil
	}
	return nil, nil
}
