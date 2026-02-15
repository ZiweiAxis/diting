package cheq

import (
	"context"
	"sync"
	"time"

	"diting/internal/models"
	"github.com/google/uuid"
)

// StubEngine 占位实现：内存存储，Create 返回对象，Submit 幂等，无真实投递。
type StubEngine struct {
	mu   sync.RWMutex
	objs map[string]*models.ConfirmationObject
}

func NewStubEngine() *StubEngine {
	return &StubEngine{objs: make(map[string]*models.ConfirmationObject)}
}

func (s *StubEngine) Create(ctx context.Context, in *CreateInput) (*models.ConfirmationObject, error) {
	id := uuid.New().String()
	obj := &models.ConfirmationObject{
		ID:              id,
		TraceID:         in.TraceID,
		Status:          models.ConfirmationStatusPending,
		CreatedAt:       time.Now(),
		ExpiresAt:       in.ExpiresAt,
		Resource:        in.Resource,
		Action:          in.Action,
		Summary:         in.Summary,
		ConfirmerIDs:    in.ConfirmerIDs,
		Type:            in.Type,
		ApprovalPolicy:  in.ApprovalPolicy,
	}
	s.mu.Lock()
	s.objs[id] = obj
	s.mu.Unlock()
	return obj, nil
}

func (s *StubEngine) GetByID(ctx context.Context, id string) (*models.ConfirmationObject, error) {
	s.mu.RLock()
	obj := s.objs[id]
	s.mu.RUnlock()
	if obj == nil {
		return nil, ErrNotFound
	}
	if time.Now().After(obj.ExpiresAt) && !obj.IsTerminal() {
		obj.Status = models.ConfirmationStatusExpired
	}
	return obj, nil
}

func (s *StubEngine) Submit(ctx context.Context, id string, approved bool, confirmerID string) error {
	s.mu.Lock()
	obj := s.objs[id]
	if obj == nil {
		s.mu.Unlock()
		return ErrNotFound
	}
	if obj.IsTerminal() {
		s.mu.Unlock()
		return ErrAlreadyProcessed
	}
	if time.Now().After(obj.ExpiresAt) {
		obj.Status = models.ConfirmationStatusExpired
		s.mu.Unlock()
		return ErrExpired
	}
	if approved {
		obj.Status = models.ConfirmationStatusApproved
	} else {
		obj.Status = models.ConfirmationStatusRejected
	}
	s.mu.Unlock()
	return nil
}
