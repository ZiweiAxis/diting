package cheq

import (
	"context"
	"fmt"
	"os"
	"time"

	"diting/internal/delivery"
	"diting/internal/models"
	"diting/internal/ownership"
	"github.com/google/uuid"
)

// EngineImpl 持久化 CHEQ：Create 时解析确认人并投递，GetByID/Submit 读写 store。
type EngineImpl struct {
	store    *JSONStore
	timeout  time.Duration
	resolve  ownership.Resolver
	delivery delivery.Provider
}

// NewEngineImpl 创建带持久化与投递的 CHEQ 引擎。
func NewEngineImpl(store *JSONStore, timeoutSeconds int, resolve ownership.Resolver, deliver delivery.Provider) *EngineImpl {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	return &EngineImpl{
		store:    store,
		timeout:  time.Duration(timeoutSeconds) * time.Second,
		resolve:  resolve,
		delivery: deliver,
	}
}

// Create 生成 ID、解析确认人、持久化、投递后返回。
func (e *EngineImpl) Create(ctx context.Context, in *CreateInput) (*models.ConfirmationObject, error) {
	if in == nil {
		return nil, fmt.Errorf("cheq: nil create input")
	}
	id := uuid.New().String()
	expiresAt := in.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(e.timeout)
	}
	confirmerIDs := in.ConfirmerIDs
	if e.resolve != nil {
		ids, _ := e.resolve.Resolve(ctx, in.Resource, in.Action)
		if len(ids) > 0 {
			confirmerIDs = ids
		}
	}
	obj := &models.ConfirmationObject{
		ID:           id,
		TraceID:      in.TraceID,
		Status:       models.ConfirmationStatusPending,
		CreatedAt:     time.Now(),
		ExpiresAt:    expiresAt,
		Resource:     in.Resource,
		Action:       in.Action,
		Summary:      in.Summary,
		ConfirmerIDs: confirmerIDs,
		Type:         in.Type,
	}
	if err := e.store.Put(ctx, obj); err != nil {
		return nil, err
	}
	if e.delivery != nil {
		opts := &delivery.DeliverOptions{ConfirmerIDs: confirmerIDs, Summary: in.Summary, ChannelType: "feishu"}
		if err := e.delivery.Deliver(ctx, &delivery.DeliverInput{Object: obj, Options: opts}); err != nil {
			fmt.Fprintf(os.Stderr, "[diting] [cheq] 飞书投递失败（请求仍待确认，可凭终端中的链接批准）: %v\n", err)
		}
	}
	return obj, nil
}

// GetByID 从 store 读取；若已过期则更新为 expired 并写回。
func (e *EngineImpl) GetByID(ctx context.Context, id string) (*models.ConfirmationObject, error) {
	obj, err := e.store.Get(ctx, id)
	if err != nil || obj == nil {
		return nil, err
	}
	if !obj.IsTerminal() && time.Now().After(obj.ExpiresAt) {
		obj.Status = models.ConfirmationStatusExpired
		_ = e.store.Put(ctx, obj)
	}
	return obj, nil
}

// Submit 幂等提交；已终态或过期返回对应错误。
func (e *EngineImpl) Submit(ctx context.Context, id string, approved bool) error {
	obj, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if obj == nil {
		return ErrNotFound
	}
	if obj.IsTerminal() {
		return ErrAlreadyProcessed
	}
	if time.Now().After(obj.ExpiresAt) {
		obj.Status = models.ConfirmationStatusExpired
		_ = e.store.Put(ctx, obj)
		return ErrExpired
	}
	if approved {
		obj.Status = models.ConfirmationStatusApproved
	} else {
		obj.Status = models.ConfirmationStatusRejected
	}
	return e.store.Put(ctx, obj)
}
