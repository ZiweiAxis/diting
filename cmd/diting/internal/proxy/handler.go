package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	"diting/internal/models"
	"github.com/google/uuid"
)

// buildRequestContext 从 HTTP 请求提取 L0 身份与 RequestContext。
// Agent 身份从 X-Agent-Token 或 Authorization 提取；Resource/Action 可从 Path 或默认值。
func buildRequestContext(r *http.Request, traceID string) *models.RequestContext {
	agentIdentity := r.Header.Get("X-Agent-Token")
	if agentIdentity == "" {
		agentIdentity = r.Header.Get("Authorization")
	}
	targetURL := r.URL.String()
	if r.URL.Scheme == "" {
		targetURL = r.Host + r.URL.RequestURI()
	}
	return &models.RequestContext{
		AgentIdentity: agentIdentity,
		Method:        r.Method,
		TargetURL:     targetURL,
		Resource:      r.URL.Path,
		Action:        r.Method,
		Headers:       r.Header.Clone(),
	}
}

// proxyHandler 处理代理请求：生成 trace_id、构建 RequestContext、走流水线。
func (s *Server) proxyHandler() http.HandlerFunc {
	upstreamURL, _ := url.Parse(s.cfg.Proxy.Upstream)
	if upstreamURL.String() == "" {
		upstreamURL, _ = url.Parse("http://localhost:8081")
	}
	rp := httputil.NewSingleHostReverseProxy(upstreamURL)
	// 保留默认 Director（负责设置 scheme/host/path/query 等），在其基础上注入 trace 头。
	origDirector := rp.Director
	rp.Director = func(outreq *http.Request) {
		if origDirector != nil {
			origDirector(outreq)
		}
		if id, ok := outreq.Context().Value(ctxKeyTraceID).(string); ok && id != "" {
			outreq.Header.Set("traceparent", id)
			outreq.Header.Set("X-Trace-ID", id)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("traceparent")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		reqCtx := buildRequestContext(r, traceID)
		ctx := context.WithValue(r.Context(), ctxKeyTraceID, traceID)
		s.pipeline.ServeHTTP(w, r.WithContext(ctx), reqCtx, rp)
	}
}

// ctxKeyTraceID 用于在 context 中存放 trace_id。
type ctxKey string

const ctxKeyTraceID ctxKey = "trace_id"
