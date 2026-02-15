// Package proxy 执行层鉴权：POST /auth/exec 与 HTTP 代理共用 Policy/CHEQ/Audit。
package proxy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"diting/internal/cheq"
	"diting/internal/delivery"
	"diting/internal/models"
)

// ExecAuthRequest 为 POST /auth/exec 的 JSON 请求体（与 proto ExecAuthRequest 对齐）。
type ExecAuthRequest struct {
	Subject     string            `json:"subject"`
	Action      string            `json:"action"`
	Resource    string            `json:"resource"`
	Context     map[string]string `json:"context,omitempty"`
	CommandLine string            `json:"command_line"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
}

// ExecAuthResponse 为 POST /auth/exec 的 JSON 响应（与 proto ExecAuthResponse 对齐）。
type ExecAuthResponse struct {
	Decision            string            `json:"decision"` // allow | deny | review
	PolicyRuleID        string            `json:"policy_rule_id,omitempty"`
	Reason              string            `json:"reason,omitempty"`
	CheqID              string            `json:"cheq_id,omitempty"`
	ApprovalTimeoutSec  int32             `json:"approval_timeout_sec,omitempty"`
	AuditMetadata      map[string]string `json:"audit_metadata,omitempty"`
}

// ExecEvaluate 对执行层请求做 L0 → Policy → allow/deny/review；review 时走 CHEQ 与投递，同步等待终态后返回 allow 或 deny。
// 与 HTTP 代理共用同一 Policy、CHEQ、DeliveryProvider、AuditStore，飞书审批逻辑一致。
func (p *pipeline) ExecEvaluate(ctx context.Context, traceID string, req *models.RequestContext) (*ExecAuthResponse, error) {
	if traceID == "" {
		traceID = "unknown"
	}

	// L0 校验
	if len(p.allowedAPIKeys) > 0 {
		token := normalizeL0Token(req.AgentIdentity)
		if token == "" {
			p.appendEvidence(ctx, traceID, req, "l0_missing", "l0", "missing or empty agent identity")
			return &ExecAuthResponse{Decision: "deny", PolicyRuleID: "l0", Reason: "missing or invalid agent identity"}, nil
		}
		if !containsString(p.allowedAPIKeys, token) {
			p.appendEvidence(ctx, traceID, req, "l0_invalid", "l0", "agent identity not in allowed list")
			return &ExecAuthResponse{Decision: "deny", PolicyRuleID: "l0", Reason: "invalid agent identity"}, nil
		}
	}

	decision, err := p.policy.Evaluate(ctx, req)
	if err != nil {
		p.appendEvidence(ctx, traceID, req, "error", "pdp_error", err.Error())
		return nil, err
	}

	timeoutSec := p.cheqTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	resource := req.Resource
	if resource == "" {
		resource = req.TargetURL
	}
	riskLevel := ""
	if req.Context != nil {
		riskLevel = req.Context["risk_level"]
	}
	var confirmerIDs []string
	approvalPolicy := ""
	if p.approvalMatcher != nil {
		m := p.approvalMatcher.Match(resource, riskLevel)
		if m.TimeoutSeconds > 0 {
			timeoutSec = m.TimeoutSeconds
		}
		confirmerIDs = m.ApprovalUserIDs
		approvalPolicy = m.ApprovalPolicy
	}

	switch {
	case decision.Allow():
		p.appendEvidence(ctx, traceID, req, "allow", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{
			Decision:     "allow",
			PolicyRuleID: decision.PolicyRuleID,
			Reason:       decision.DecisionReason,
		}, nil
	case decision.Deny():
		p.appendEvidence(ctx, traceID, req, "deny", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{
			Decision:     "deny",
			PolicyRuleID: decision.PolicyRuleID,
			Reason:       decision.DecisionReason,
		}, nil
	case decision.Review():
		expiresAt := time.Now().Add(time.Duration(timeoutSec) * time.Second)
		summary := req.TargetURL
		if summary == "" {
			summary = req.Action + " " + req.Resource
		}
		in := &cheq.CreateInput{
			TraceID:        traceID,
			Resource:      resource,
			Action:        req.Action,
			Summary:       summary,
			ExpiresAt:     expiresAt,
			ConfirmerIDs:  confirmerIDs,
			Type:          "operation_approval",
			ApprovalPolicy: approvalPolicy,
		}
		obj, err := p.cheq.Create(ctx, in)
		if err != nil {
			p.appendEvidence(ctx, traceID, req, "review_error", "cheq_create", err.Error())
			return nil, err
		}
		if !p.reviewRequiresApproval {
			_ = p.cheq.Submit(ctx, obj.ID, true, "")
			p.appendEvidenceWithCHEQ(ctx, traceID, req, "approved", decision.PolicyRuleID, decision.DecisionReason, string(models.ConfirmationStatusApproved), obj.ConfirmerIDs)
			return &ExecAuthResponse{
				Decision:           "allow",
				PolicyRuleID:       decision.PolicyRuleID,
				Reason:             decision.DecisionReason,
				CheqID:             obj.ID,
				ApprovalTimeoutSec: int32(timeoutSec),
			}, nil
		}
		_, _ = fmt.Fprintf(os.Stderr, "[diting] [exec] CHEQ 待确认 id=%s 批准: http://localhost:8080/cheq/approve?id=%s&approved=true 拒绝: http://localhost:8080/cheq/approve?id=%s&approved=false\n", obj.ID, obj.ID, obj.ID)
		deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
		var finalStatus string
		var reminded bool
		var o *models.ConfirmationObject
		for time.Now().Before(deadline) {
			o, _ = p.cheq.GetByID(ctx, obj.ID)
			if o == nil {
				time.Sleep(2 * time.Second)
				continue
			}
			if o.IsTerminal() {
				finalStatus = string(o.Status)
				break
			}
			remindSec := p.reminderSecondsBeforeTimeout
			if remindSec <= 0 {
				remindSec = 60
			}
			if !reminded && p.delivery != nil && time.Until(o.ExpiresAt) <= time.Duration(remindSec)*time.Second {
				reminded = true
				_ = p.delivery.Deliver(ctx, &delivery.DeliverInput{Object: o, Options: &delivery.DeliverOptions{Summary: "【提醒】该请求即将超时，请尽快处理"}})
			}
			time.Sleep(2 * time.Second)
		}
		var confirmerIDs []string
		if o != nil {
			confirmerIDs = o.ConfirmerIDs
		}
		if finalStatus == string(models.ConfirmationStatusApproved) {
			p.appendEvidenceWithCHEQ(ctx, traceID, req, "approved", decision.PolicyRuleID, decision.DecisionReason, finalStatus, confirmerIDs)
			return &ExecAuthResponse{
				Decision:           "allow",
				PolicyRuleID:       decision.PolicyRuleID,
				Reason:             decision.DecisionReason,
				CheqID:             obj.ID,
				ApprovalTimeoutSec: int32(timeoutSec),
			}, nil
		}
		if finalStatus == "" {
			finalStatus = "expired"
		}
		p.appendEvidenceWithCHEQ(ctx, traceID, req, finalStatus, decision.PolicyRuleID, decision.DecisionReason, finalStatus, confirmerIDs)
		return &ExecAuthResponse{
			Decision:           "deny",
			PolicyRuleID:       decision.PolicyRuleID,
			Reason:             "confirmation " + finalStatus,
			CheqID:             obj.ID,
			ApprovalTimeoutSec: int32(timeoutSec),
		}, nil
	default:
		p.appendEvidence(ctx, traceID, req, "unknown", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{
			Decision:     "deny",
			PolicyRuleID: decision.PolicyRuleID,
			Reason:       decision.DecisionReason,
		}, nil
	}
}

// ReviewAuditInfo 在 review 非阻塞返回时携带，用于 CHEQ 终态后写审计（AuthStream approval_push 路径）。
type ReviewAuditInfo struct {
	PolicyRuleID   string
	DecisionReason string
}

// ExecEvaluateNonBlocking 与 ExecEvaluate 相同，但 review 时仅创建 CHEQ 并立即返回 decision=review、cheq_id，不轮询等待。
// 用于 AuthStream：调用方在收到 review 后轮询 GetByID，终态时写审计并推送 approval_push。
func (p *pipeline) ExecEvaluateNonBlocking(ctx context.Context, traceID string, req *models.RequestContext) (*ExecAuthResponse, *ReviewAuditInfo, error) {
	if traceID == "" {
		traceID = "unknown"
	}
	if len(p.allowedAPIKeys) > 0 {
		token := normalizeL0Token(req.AgentIdentity)
		if token == "" {
			p.appendEvidence(ctx, traceID, req, "l0_missing", "l0", "missing or empty agent identity")
			return &ExecAuthResponse{Decision: "deny", PolicyRuleID: "l0", Reason: "missing or invalid agent identity"}, nil, nil
		}
		if !containsString(p.allowedAPIKeys, token) {
			p.appendEvidence(ctx, traceID, req, "l0_invalid", "l0", "agent identity not in allowed list")
			return &ExecAuthResponse{Decision: "deny", PolicyRuleID: "l0", Reason: "invalid agent identity"}, nil, nil
		}
	}
	decision, err := p.policy.Evaluate(ctx, req)
	if err != nil {
		p.appendEvidence(ctx, traceID, req, "error", "pdp_error", err.Error())
		return nil, nil, err
	}
	timeoutSec := p.cheqTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	nbResource := req.Resource
	if nbResource == "" {
		nbResource = req.TargetURL
	}
	nbRiskLevel := ""
	if req.Context != nil {
		nbRiskLevel = req.Context["risk_level"]
	}
	var nbConfirmerIDs []string
	nbApprovalPolicy := ""
	if p.approvalMatcher != nil {
		m := p.approvalMatcher.Match(nbResource, nbRiskLevel)
		if m.TimeoutSeconds > 0 {
			timeoutSec = m.TimeoutSeconds
		}
		nbConfirmerIDs = m.ApprovalUserIDs
		nbApprovalPolicy = m.ApprovalPolicy
	}
	switch {
	case decision.Allow():
		p.appendEvidence(ctx, traceID, req, "allow", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{Decision: "allow", PolicyRuleID: decision.PolicyRuleID, Reason: decision.DecisionReason}, nil, nil
	case decision.Deny():
		p.appendEvidence(ctx, traceID, req, "deny", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{Decision: "deny", PolicyRuleID: decision.PolicyRuleID, Reason: decision.DecisionReason}, nil, nil
	case decision.Review():
		expiresAt := time.Now().Add(time.Duration(timeoutSec) * time.Second)
		summary := req.TargetURL
		if summary == "" {
			summary = req.Action + " " + req.Resource
		}
		in := &cheq.CreateInput{
			TraceID:        traceID,
			Resource:       nbResource,
			Action:         req.Action,
			Summary:        summary,
			ExpiresAt:      expiresAt,
			ConfirmerIDs:   nbConfirmerIDs,
			Type:           "operation_approval",
			ApprovalPolicy: nbApprovalPolicy,
		}
		obj, err := p.cheq.Create(ctx, in)
		if err != nil {
			p.appendEvidence(ctx, traceID, req, "review_error", "cheq_create", err.Error())
			return nil, nil, err
		}
		if !p.reviewRequiresApproval {
			_ = p.cheq.Submit(ctx, obj.ID, true, "")
			p.appendEvidenceWithCHEQ(ctx, traceID, req, "approved", decision.PolicyRuleID, decision.DecisionReason, string(models.ConfirmationStatusApproved), obj.ConfirmerIDs)
			return &ExecAuthResponse{
				Decision: "allow", PolicyRuleID: decision.PolicyRuleID, Reason: decision.DecisionReason,
				CheqID: obj.ID, ApprovalTimeoutSec: int32(timeoutSec),
			}, nil, nil
		}
		_, _ = fmt.Fprintf(os.Stderr, "[diting] [authstream] CHEQ 待确认 id=%s\n", obj.ID)
		return &ExecAuthResponse{
			Decision: "review", PolicyRuleID: decision.PolicyRuleID, Reason: decision.DecisionReason,
			CheqID: obj.ID, ApprovalTimeoutSec: int32(timeoutSec),
		}, &ReviewAuditInfo{PolicyRuleID: decision.PolicyRuleID, DecisionReason: decision.DecisionReason}, nil
	default:
		p.appendEvidence(ctx, traceID, req, "unknown", decision.PolicyRuleID, decision.DecisionReason)
		return &ExecAuthResponse{Decision: "deny", PolicyRuleID: decision.PolicyRuleID, Reason: decision.DecisionReason}, nil, nil
	}
}

// RecordCHEQDecision 在 CHEQ 终态后写审计（AuthStream 在推送 approval_push 前调用）。
func (p *pipeline) RecordCHEQDecision(ctx context.Context, traceID string, req *models.RequestContext, policyRuleID, decisionReason, cheqID, finalStatus string, confirmerIDs []string) {
	p.appendEvidenceWithCHEQ(ctx, traceID, req, finalStatus, policyRuleID, decisionReason, finalStatus, confirmerIDs)
}

// GetCHEQByID 供 AuthStream 轮询 CHEQ 状态（封装 cheq.Engine.GetByID）。
func (p *pipeline) GetCHEQByID(ctx context.Context, id string) (*models.ConfirmationObject, error) {
	return p.cheq.GetByID(ctx, id)
}

// BuildRequestContextFromExec 从 ExecAuthRequest 构建 RequestContext，供策略与审计复用。
func BuildRequestContextFromExec(in *ExecAuthRequest, agentIdentity string) *models.RequestContext {
	if in == nil {
		return nil
	}
	action := in.Action
	if action != "" && !strings.HasPrefix(action, "exec:") {
		action = "exec:" + action
	}
	return &models.RequestContext{
		AgentIdentity: agentIdentity,
		Method:        "EXEC",
		TargetURL:     in.CommandLine,
		Resource:      in.Resource,
		Action:        action,
		Headers:       nil,
		Context:       in.Context,
	}
}
