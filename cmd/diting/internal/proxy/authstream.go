// Package proxy 实现 AuthStream 长连接（Story 8.4）：WebSocket 握手、鉴权请求、异步 approval_push。
package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"diting/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// AuthStreamRequest 客户端消息（与 proto AuthStreamRequest 对齐）。
type AuthStreamRequest struct {
	RequestID string                 `json:"request_id"`
	Init      *AuthStreamInit         `json:"init,omitempty"`
	Auth      *ExecAuthRequest        `json:"auth,omitempty"`
	Ping      string                  `json:"ping,omitempty"`
}

// AuthStreamInit 握手包。
type AuthStreamInit struct {
	ClientID    string `json:"client_id"`
	Resource    string `json:"resource"`
	AgentVersion string `json:"agent_version,omitempty"`
}

// AuthStreamResponse 服务端消息。
type AuthStreamResponse struct {
	RequestID    string               `json:"request_id,omitempty"`
	Immediate    *ExecAuthResponse    `json:"immediate,omitempty"`
	ApprovalPush *AuthStreamApprovalPush `json:"approval_push,omitempty"`
	ProfileUpdate *SandboxProfile     `json:"profile_update,omitempty"`
	Pong         string               `json:"pong,omitempty"`
}

// AuthStreamApprovalPush 异步审批结果推送。
type AuthStreamApprovalPush struct {
	CheqID        string `json:"cheq_id"`
	FinalDecision string `json:"final_decision"` // allow | deny
	Reason        string `json:"reason,omitempty"`
}

func (s *Server) authStreamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		ctx := r.Context()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var req AuthStreamRequest
			if err := json.Unmarshal(data, &req); err != nil {
				_ = sendStreamResp(conn, req.RequestID, nil, nil, nil, "invalid json")
				continue
			}
			if req.RequestID == "" {
				req.RequestID = uuid.New().String()
			}
			if req.Init != nil {
				_ = sendStreamResp(conn, req.RequestID, nil, nil, nil, "pong")
				continue
			}
			if req.Ping != "" {
				_ = sendStreamResp(conn, req.RequestID, nil, nil, nil, "pong")
				continue
			}
			if req.Auth != nil {
				traceID := req.Auth.TraceID
				if traceID == "" {
					traceID = uuid.New().String()
				}
				agentIdentity := req.Auth.Subject
				reqCtx := BuildRequestContextFromExec(req.Auth, agentIdentity)
				if reqCtx == nil {
					_ = sendStreamResp(conn, req.RequestID, &ExecAuthResponse{Decision: "deny", Reason: "missing subject/action/resource"}, nil, nil, "")
					continue
				}
				resp, auditInfo, err := s.pipeline.ExecEvaluateNonBlocking(ctx, traceID, reqCtx)
				if err != nil {
					_ = sendStreamResp(conn, req.RequestID, &ExecAuthResponse{Decision: "deny", Reason: "evaluate failed"}, nil, nil, "")
					continue
				}
				_ = sendStreamResp(conn, req.RequestID, resp, nil, nil, "")
				if resp != nil && resp.Decision == "review" && resp.CheqID != "" && auditInfo != nil {
					go waitAndPushApproval(ctx, s.pipeline, conn, req.RequestID, traceID, reqCtx, resp.CheqID, resp.ApprovalTimeoutSec, auditInfo.PolicyRuleID, auditInfo.DecisionReason)
				}
			}
		}
	}
}

func sendStreamResp(conn *websocket.Conn, requestID string, immediate *ExecAuthResponse, approvalPush *AuthStreamApprovalPush, profileUpdate *SandboxProfile, pong string) error {
	out := AuthStreamResponse{RequestID: requestID}
	if immediate != nil {
		out.Immediate = immediate
	}
	if approvalPush != nil {
		out.ApprovalPush = approvalPush
	}
	if profileUpdate != nil {
		out.ProfileUpdate = profileUpdate
	}
	if pong != "" {
		out.Pong = pong
	}
	return conn.WriteJSON(out)
}

func waitAndPushApproval(ctx context.Context, pl *pipeline, conn *websocket.Conn, requestID, traceID string, reqCtx *models.RequestContext, cheqID string, timeoutSec int32, policyRuleID, decisionReason string) {
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	var finalStatus string
	var confirmerIDs []string
	for time.Now().Before(deadline) {
		o, _ := pl.GetCHEQByID(ctx, cheqID)
		if o != nil && o.IsTerminal() {
			finalStatus = string(o.Status)
			confirmerIDs = o.ConfirmerIDs
			break
		}
		time.Sleep(2 * time.Second)
	}
	if finalStatus == "" {
		finalStatus = "expired"
	}
	pl.RecordCHEQDecision(ctx, traceID, reqCtx, policyRuleID, decisionReason, cheqID, finalStatus, confirmerIDs)
	decision := "deny"
	if finalStatus == string(models.ConfirmationStatusApproved) {
		decision = "allow"
	}
	_ = sendStreamResp(conn, requestID, nil, &AuthStreamApprovalPush{
		CheqID:        cheqID,
		FinalDecision: decision,
		Reason:        "confirmation " + finalStatus,
	}, nil, "")
}
