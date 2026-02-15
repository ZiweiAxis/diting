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
	store         *JSONStore
	timeout       time.Duration
	resolve       ownership.Resolver
	delivery      delivery.Provider
	approvalPolicy string // "any" 或 "all"（I-008）
}

// NewEngineImpl 创建带持久化与投递的 CHEQ 引擎。approvalPolicy 为 "any"（任一通过）或 "all"（全部通过），空则按 "any"。
func NewEngineImpl(store *JSONStore, timeoutSeconds int, resolve ownership.Resolver, deliver delivery.Provider, approvalPolicy string) *EngineImpl {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	if approvalPolicy != "all" {
		approvalPolicy = "any"
	}
	return &EngineImpl{
		store:          store,
		timeout:        time.Duration(timeoutSeconds) * time.Second,
		resolve:        resolve,
		delivery:       deliver,
		approvalPolicy: approvalPolicy,
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
	policy := e.approvalPolicy
	if in.ApprovalPolicy == "all" {
		policy = "all"
	} else if in.ApprovalPolicy != "" {
		policy = "any"
	}
	obj := &models.ConfirmationObject{
		ID:              id,
		TraceID:         in.TraceID,
		Status:          models.ConfirmationStatusPending,
		CreatedAt:       time.Now(),
		ExpiresAt:       expiresAt,
		Resource:        in.Resource,
		Action:          in.Action,
		Summary:         in.Summary,
		ConfirmerIDs:    confirmerIDs,
		Type:            in.Type,
		ApprovalPolicy:  policy,
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

// Submit 幂等提交；已终态或过期返回对应错误。confirmerID 用于「全部通过」时记录谁批准。
func (e *EngineImpl) Submit(ctx context.Context, id string, approved bool, confirmerID string) error {
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
	if !approved {
		obj.Status = models.ConfirmationStatusRejected
		return e.store.Put(ctx, obj)
	}
	// approved == true
	policy := obj.ApprovalPolicy
	if policy == "" {
		policy = e.approvalPolicy
	}
	if policy == "all" {
		if obj.ApprovedBy == nil {
			obj.ApprovedBy = []string{}
		}
		already := false
		for _, x := range obj.ApprovedBy {
			if x == confirmerID {
				already = true
				break
			}
		}
		if !already && confirmerID != "" {
			obj.ApprovedBy = append(obj.ApprovedBy, confirmerID)
		}
		if len(obj.ApprovedBy) >= len(obj.ConfirmerIDs) {
			obj.Status = models.ConfirmationStatusApproved
		}
		// 未达全部时仅写回 ApprovedBy，不设终态
		return e.store.Put(ctx, obj)
	}
	// any: 任一通过即放行
	obj.Status = models.ConfirmationStatusApproved
	return e.store.Put(ctx, obj)
}
