package audit

import (
	"context"

	"diting/internal/models"
)

// StubStore 占位实现：Append 与 QueryByTraceID 为内存/无操作，供 Phase 2 装配。
type StubStore struct {
	evidences []*models.Evidence
}

func NewStubStore() *StubStore {
	return &StubStore{evidences: make([]*models.Evidence, 0)}
}

func (s *StubStore) Append(ctx context.Context, e *models.Evidence) error {
	s.evidences = append(s.evidences, e)
	return nil
}

func (s *StubStore) QueryByTraceID(ctx context.Context, traceID string) ([]*models.Evidence, error) {
	var out []*models.Evidence
	for _, e := range s.evidences {
		if e.TraceID == traceID {
			out = append(out, e)
		}
	}
	return out, nil
}
