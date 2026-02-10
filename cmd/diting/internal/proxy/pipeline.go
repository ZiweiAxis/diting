package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/delivery"
	"diting/internal/models"
	"diting/internal/policy"
)

// responseWriterWithTraceID 在首次 WriteHeader 时注入 X-Trace-ID，便于验收时按 trace_id 查审计。
type responseWriterWithTraceID struct {
	http.ResponseWriter
	traceID string
	wrote   bool
}

func (w *responseWriterWithTraceID) WriteHeader(code int) {
	if !w.wrote {
		w.Header().Set("X-Trace-ID", w.traceID)
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

// pipeline 封装 L0 → PDP → allow/deny/review → 审计的流水线。
type pipeline struct {
	policy                 policy.Engine
	cheq                   cheq.Engine
	audit                  audit.Store
	delivery               delivery.Provider
	cheqTimeoutSec         int
	reviewRequiresApproval bool
	allowedAPIKeys         []string // 非空时启用 L0 校验：身份须在此列表中
}

func (p *pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request, reqCtx *models.RequestContext, rp *httputil.ReverseProxy) {
	ctx := r.Context()
	traceID, _ := ctx.Value(ctxKeyTraceID).(string)
	if traceID == "" {
		traceID = "unknown"
	}

	wrap := &responseWriterWithTraceID{ResponseWriter: w, traceID: traceID}

	// 3.2.1 L0 校验：若配置了 allowed_api_keys，未携带或无效身份则拒绝并写审计。
	if len(p.allowedAPIKeys) > 0 {
		token := normalizeL0Token(reqCtx.AgentIdentity)
		if token == "" {
			p.appendEvidence(ctx, traceID, reqCtx, "l0_missing", "l0", "missing or empty agent identity")
			wrap.WriteHeader(http.StatusUnauthorized)
			_, _ = wrap.Write([]byte("missing or invalid agent identity"))
			return
		}
		if !containsString(p.allowedAPIKeys, token) {
			p.appendEvidence(ctx, traceID, reqCtx, "l0_invalid", "l0", "agent identity not in allowed list")
			wrap.WriteHeader(http.StatusUnauthorized)
			_, _ = wrap.Write([]byte("invalid agent identity"))
			return
		}
	}

	// 3.2.2 调用 PolicyEngine.Evaluate
	decision, err := p.policy.Evaluate(ctx, reqCtx)
	if err != nil {
		p.appendEvidence(ctx, traceID, reqCtx, "error", "pdp_error", err.Error())
		wrap.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch {
	case decision.Allow():
		// 3.2.3 allow：转发后写审计
		rp.ServeHTTP(wrap, r)
		p.appendEvidence(ctx, traceID, reqCtx, "allow", decision.PolicyRuleID, decision.DecisionReason)
	case decision.Deny():
		// 3.2.4 deny：拒绝并写审计
		p.appendEvidence(ctx, traceID, reqCtx, "deny", decision.PolicyRuleID, decision.DecisionReason)
		wrap.WriteHeader(http.StatusForbidden)
		_, _ = wrap.Write([]byte(decision.DecisionReason))
	case decision.Review():
		// 3.2.5 review：创建 CHEQ；若需人工确认则轮询直到终态或超时，否则立即放行（占位）
		timeoutSec := p.cheqTimeoutSec
		if timeoutSec <= 0 {
			timeoutSec = 300
		}
		expiresAt := time.Now().Add(time.Duration(timeoutSec) * time.Second)
		in := &cheq.CreateInput{
			TraceID:   traceID,
			Resource:  reqCtx.Resource,
			Action:    reqCtx.Action,
			Summary:   reqCtx.TargetURL,
			ExpiresAt: expiresAt,
			Type:      "operation_approval",
		}
		obj, err := p.cheq.Create(ctx, in)
		if err != nil {
			p.appendEvidence(ctx, traceID, reqCtx, "review_error", "cheq_create", err.Error())
			wrap.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !p.reviewRequiresApproval {
			_ = p.cheq.Submit(ctx, obj.ID, true)
			rp.ServeHTTP(wrap, r)
			p.appendEvidenceWithCHEQ(ctx, traceID, reqCtx, "approved", decision.PolicyRuleID, decision.DecisionReason, string(models.ConfirmationStatusApproved), obj.ConfirmerIDs)
			break
		}
		_, _ = fmt.Fprintf(os.Stderr, "[diting] CHEQ 待确认 id=%s 批准: http://localhost:8080/cheq/approve?id=%s&approved=true 拒绝: http://localhost:8080/cheq/approve?id=%s&approved=false\n", obj.ID, obj.ID, obj.ID)
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
			if !reminded && p.delivery != nil && time.Until(o.ExpiresAt) <= 60*time.Second {
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
			rp.ServeHTTP(wrap, r)
			p.appendEvidenceWithCHEQ(ctx, traceID, reqCtx, "approved", decision.PolicyRuleID, decision.DecisionReason, finalStatus, confirmerIDs)
		} else {
			wrap.WriteHeader(http.StatusForbidden)
			if finalStatus == "" {
				finalStatus = "expired"
			}
			p.appendEvidenceWithCHEQ(ctx, traceID, reqCtx, finalStatus, decision.PolicyRuleID, decision.DecisionReason, finalStatus, confirmerIDs)
			_, _ = wrap.Write([]byte("confirmation " + finalStatus))
		}
	default:
		p.appendEvidence(ctx, traceID, reqCtx, "unknown", decision.PolicyRuleID, decision.DecisionReason)
		wrap.WriteHeader(http.StatusForbidden)
	}
}

// normalizeL0Token 去掉 Authorization 的 "Bearer " 前缀，便于与配置的 key 比对。
func normalizeL0Token(identity string) string {
	s := strings.TrimSpace(identity)
	if strings.HasPrefix(s, "Bearer ") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "Bearer "))
	}
	return s
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func (p *pipeline) appendEvidence(ctx context.Context, traceID string, req *models.RequestContext, decision, policyRuleID, reason string) {
	p.appendEvidenceWithCHEQ(ctx, traceID, req, decision, policyRuleID, reason, "", nil)
}

func (p *pipeline) appendEvidenceWithCHEQ(ctx context.Context, traceID string, req *models.RequestContext, decision, policyRuleID, reason, cheqStatus string, confirmerIDs []string) {
	confirmer := ""
	if len(confirmerIDs) > 0 {
		confirmer = strings.Join(confirmerIDs, ",")
	}
	_ = p.audit.Append(ctx, &models.Evidence{
		TraceID:        traceID,
		AgentID:        req.AgentIdentity,
		PolicyRuleID:   policyRuleID,
		DecisionReason: reason,
		Decision:       decision,
		CHEQStatus:     cheqStatus,
		Confirmer:      confirmer,
		Timestamp:      time.Now(),
		Resource:       req.Resource,
		Action:         req.Action,
	})
}
