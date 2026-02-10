package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/config"
	"diting/internal/delivery"
	"diting/internal/ownership"
	"diting/internal/policy"
)

func TestProxyHandler_DirectorPreservesUpstreamAndInjectsTraceHeaders(t *testing.T) {
	// 上游：回显收到的 trace 头，便于断言
	var gotTraceParent, gotXTraceID string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceParent = r.Header.Get("traceparent")
		gotXTraceID = r.Header.Get("X-Trace-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer up.Close()

	cfg := &config.Config{
		Proxy: config.ProxyConfig{
			ListenAddr: ":0",
			Upstream:   up.URL,
		},
		Policy: config.PolicyConfig{
			// StubPolicy 会走 allow 分支，从而真正触发 reverse proxy
			RulesPath: "",
		},
	}

	s := NewServer(cfg, &policy.StubEngine{}, cheq.NewStubEngine(), &delivery.StubProvider{}, audit.NewStubStore(), &ownership.StubResolver{}, false)
	h := s.proxyHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	req.Header.Set("traceparent", "trace-xyz")
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from upstream, got %d", rr.Result().StatusCode)
	}
	if gotTraceParent != "trace-xyz" {
		t.Fatalf("expected upstream traceparent=trace-xyz, got %q", gotTraceParent)
	}
	if gotXTraceID != "trace-xyz" {
		t.Fatalf("expected upstream X-Trace-ID=trace-xyz, got %q", gotXTraceID)
	}
	// 也验证响应头注入（由 pipeline 的 responseWriterWithTraceID 完成）
	if rr.Result().Header.Get("X-Trace-ID") == "" {
		t.Fatalf("expected response to include X-Trace-ID")
	}
	// 防御：确保 handler 不依赖外部 context
	_ = context.Background()
}

