package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/http/httptest"
	"net/url"
	"testing"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/models"
	"diting/internal/policy"
)

func TestPipelineAllowWritesAudit(t *testing.T) {
	// 使用 httptest 作为上游，避免 ReverseProxy 挂起
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstreamServer.Close()

	store := audit.NewStubStore()
	pl := &pipeline{
		policy: &policy.StubEngine{},
		cheq:   cheq.NewStubEngine(),
		audit:  store,
	}
	upstreamURL, _ := url.Parse(upstreamServer.URL)
	rp := httputil.NewSingleHostReverseProxy(upstreamURL)

	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxKeyTraceID, "trace-1"))
	reqCtx := &models.RequestContext{AgentIdentity: "agent1", Method: "GET", Resource: "/foo", Action: "GET"}

	rec := httptest.NewRecorder()
	pl.ServeHTTP(rec, req, reqCtx, rp)

	// Stub 策略恒 Allow，应写一条 allow 审计
	evs, _ := store.QueryByTraceID(context.Background(), "trace-1")
	if len(evs) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(evs))
	}
	if evs[0].Decision != "allow" {
		t.Errorf("expected decision allow, got %s", evs[0].Decision)
	}
	if evs[0].TraceID != "trace-1" {
		t.Errorf("expected trace_id trace-1, got %s", evs[0].TraceID)
	}
	if evs[0].PolicyRuleID == "" {
		t.Error("policy_rule_id should be set")
	}
}
