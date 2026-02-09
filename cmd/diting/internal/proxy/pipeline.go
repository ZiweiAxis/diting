package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
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
}

func (p *pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request, reqCtx *models.RequestContext, rp *httputil.ReverseProxy) {
	ctx := r.Context()
	traceID, _ := ctx.Value(ctxKeyTraceID).(string)
	if traceID == "" {
		traceID = "unknown"
	}

	// 3.2.1 L0 校验（可选）：此处不强制拒绝无身份请求，仅记录。
	// 3.2.2 调用 PolicyEngine.Evaluate
	decision, err := p.policy.Evaluate(ctx, reqCtx)
	if err != nil {
		p.appendEvidence(ctx, traceID, reqCtx, "error", "pdp_error", err.Error())
		wrap := &responseWriterWithTraceID{ResponseWriter: w, traceID: traceID}
		wrap.WriteHeader(http.StatusInternalServerError)
		return
	}

	wrap := &responseWriterWithTraceID{ResponseWriter: w, traceID: traceID}
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
			p.appendEvidence(ctx, traceID, reqCtx, "approved", decision.PolicyRuleID, decision.DecisionReason)
			break
		}
		_, _ = fmt.Fprintf(os.Stderr, "[diting] CHEQ 待确认 id=%s 批准: http://localhost:8080/cheq/approve?id=%s&approved=true 拒绝: http://localhost:8080/cheq/approve?id=%s&approved=false\n", obj.ID, obj.ID, obj.ID)
		deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
		var finalStatus string
		var reminded bool
		for time.Now().Before(deadline) {
			o, _ := p.cheq.GetByID(ctx, obj.ID)
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
		if finalStatus == string(models.ConfirmationStatusApproved) {
			rp.ServeHTTP(wrap, r)
			p.appendEvidence(ctx, traceID, reqCtx, "approved", decision.PolicyRuleID, decision.DecisionReason)
		} else {
			wrap.WriteHeader(http.StatusForbidden)
			if finalStatus == "" {
				finalStatus = "expired"
			}
			p.appendEvidence(ctx, traceID, reqCtx, finalStatus, decision.PolicyRuleID, decision.DecisionReason)
			_, _ = wrap.Write([]byte("confirmation " + finalStatus))
		}
	default:
		p.appendEvidence(ctx, traceID, reqCtx, "unknown", decision.PolicyRuleID, decision.DecisionReason)
		wrap.WriteHeader(http.StatusForbidden)
	}
}

func (p *pipeline) appendEvidence(ctx context.Context, traceID string, req *models.RequestContext, decision, policyRuleID, reason string) {
	_ = p.audit.Append(ctx, &models.Evidence{
		TraceID:        traceID,
		AgentID:        req.AgentIdentity,
		PolicyRuleID:   policyRuleID,
		DecisionReason: reason,
		Decision:       decision,
		Timestamp:      time.Now(),
		Resource:       req.Resource,
		Action:         req.Action,
	})
}
