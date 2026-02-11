// Package proxy 提供反向代理与请求流水线；Phase 2 仅启动探针与占位监听。
package proxy

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/config"
	"diting/internal/delivery"
	"diting/internal/ownership"
	"diting/internal/policy"
)

// Server 持有策略、CHEQ、投递、审计、归属接口，并暴露探针与代理端口。
type Server struct {
	cfg      *config.Config
	policy   policy.Engine
	cheq     cheq.Engine
	delivery delivery.Provider
	audit    audit.Store
	ownership ownership.Resolver
	pipeline *pipeline
}

// NewServer 构造 Server；各组件由调用方注入。reviewRequiresApproval 为 true 时 review 路径轮询等待确认，否则立即放行（占位行为）。
func NewServer(
	cfg *config.Config,
	policy policy.Engine,
	cheq cheq.Engine,
	delivery delivery.Provider,
	audit audit.Store,
	ownership ownership.Resolver,
	reviewRequiresApproval bool,
) *Server {
	return &Server{
		cfg:       cfg,
		policy:    policy,
		cheq:      cheq,
		delivery:  delivery,
		audit:     audit,
		ownership: ownership,
		pipeline:  &pipeline{policy: policy, cheq: cheq, audit: audit, delivery: delivery, cheqTimeoutSec: cfg.CHEQ.TimeoutSeconds, reviewRequiresApproval: reviewRequiresApproval, allowedAPIKeys: cfg.Proxy.AllowedAPIKeys},
	}
}

// Handler 返回用于注册路由的 HTTP Handler，供测试或外部嵌入使用。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/debug/audit", s.debugAuditHandler())
	mux.HandleFunc("/cheq/approve", s.cheqApproveHandler())
	mux.HandleFunc("/feishu/card", s.feishuCardHandler())
	mux.HandleFunc("/auth/exec", s.execAuthHandler())
	mux.HandleFunc("/auth/sandbox-profile", s.sandboxProfileHandler())
	mux.HandleFunc("/auth/stream", s.authStreamHandler())
	mux.Handle("/", s.proxyHandler())
	return mux
}

// Serve 启动 HTTP 服务：/healthz、/readyz 与代理监听（Phase 2 代理先返回 503）。
func (s *Server) Serve(ctx context.Context) error {
	addr := s.cfg.Proxy.ListenAddr
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	return server.ListenAndServe()
}

// debugAuditHandler 返回 GET /debug/audit?trace_id=xxx 的 JSON 结果，供验收「审计可查」。
func (s *Server) debugAuditHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		traceID := r.URL.Query().Get("trace_id")
		if traceID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"missing trace_id"}`))
			return
		}
		list, err := s.audit.QueryByTraceID(r.Context(), traceID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"query failed"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	}
}

// cheqApproveHandler 处理 GET/POST /cheq/approve?id=xxx&approved=true|false，用于人工确认后提交。
func (s *Server) cheqApproveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		approvedStr := r.URL.Query().Get("approved")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"missing id"}`))
			return
		}
		approved := approvedStr == "true" || approvedStr == "1" || approvedStr == "yes"
		err := s.cheq.Submit(r.Context(), id, approved)
		if err != nil {
			if err == cheq.ErrNotFound {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"not found"}`))
				return
			}
			if err == cheq.ErrAlreadyProcessed || err == cheq.ErrExpired {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"already processed or expired"}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"submit failed"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		approvedJSON := "false"
		if approved {
			approvedJSON = "true"
		}
		_, _ = w.Write([]byte(`{"ok":true,"approved":` + approvedJSON + `}`))
	}
}

// execAuthHandler 处理 POST /auth/exec 执行能力鉴权；与 HTTP 代理共用 Policy、CHEQ、Audit（Story 7.1、7.2）。
func (s *Server) execAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body ExecAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid json"}`))
			return
		}
		traceID := body.TraceID
		if traceID == "" {
			traceID = r.Header.Get("traceparent")
		}
		if traceID == "" {
			traceID = r.Header.Get("X-Trace-ID")
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}
		agentIdentity := r.Header.Get("X-Agent-Token")
		if agentIdentity == "" {
			agentIdentity = r.Header.Get("Authorization")
		}
		if body.Subject != "" {
			agentIdentity = body.Subject
		}
		reqCtx := BuildRequestContextFromExec(&body, agentIdentity)
		if reqCtx == nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"missing subject/action/resource"}`))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyTraceID, traceID)
		resp, err := s.pipeline.ExecEvaluate(ctx, traceID, reqCtx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"evaluate failed"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-ID", traceID)
		if resp.Decision == "allow" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// feishuCardHandler 处理飞书卡片回调 POST /feishu/card（HTTP 回调方式）。长连接方式下卡片点击事件在 feishu.RunLongConnection 中处理。
func (s *Server) feishuCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var callback map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		action, _ := callback["action"].(map[string]interface{})
		value, _ := action["value"].(map[string]interface{})
		if value == nil {
			if vs, ok := action["value"].(string); ok && vs != "" {
				var vm map[string]interface{}
				if json.Unmarshal([]byte(vs), &vm) == nil {
					value = vm
				}
			}
		}
		if value == nil {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"toast":{"type":"info","content":"忽略"}}`))
			return
		}
		requestID, _ := value["request_id"].(string)
		actionType, _ := value["action"].(string)
		if requestID == "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"toast":{"type":"info","content":"缺少 request_id"}}`))
			return
		}
		approved := actionType == "approve"
		err := s.cheq.Submit(r.Context(), requestID, approved)
		if err != nil {
			if err == cheq.ErrNotFound || err == cheq.ErrExpired || err == cheq.ErrAlreadyProcessed {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"toast":{"type":"warning","content":"该请求已失效或已处理"}}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		msg := "已拒绝"
		if approved {
			msg = "已批准"
		}
		_, _ = w.Write([]byte(`{"toast":{"type":"success","content":"` + msg + `"}}`))
	}
}
