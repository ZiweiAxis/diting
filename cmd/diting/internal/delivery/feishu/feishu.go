package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"diting/internal/config"
	"diting/internal/delivery"
)

const (
	tokenAPI   = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	messageAPI = "https://open.feishu.cn/open-apis/im/v1/messages"
)

// Provider 飞书投递：获取 tenant_access_token，向用户或群发送待确认消息。
type Provider struct {
	cfg     config.FeishuConfig
	client  *http.Client
	mu      sync.RWMutex
	token   string
	expiry  time.Time
}

// NewProvider 根据飞书配置创建；app_secret 应从环境变量读取（config.Load 已做 env 覆盖）。
func NewProvider(cfg config.FeishuConfig) *Provider {
	return &Provider{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Deliver 将待确认对象以文本消息发送到飞书（优先 approval_user_id，否则 chat_id）。
func (p *Provider) Deliver(ctx context.Context, in *delivery.DeliverInput) error {
	if in == nil || in.Object == nil {
		return fmt.Errorf("feishu: nil object")
	}
	if !p.cfg.Enabled || p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		return fmt.Errorf("feishu: not enabled or missing app_id/app_secret")
	}
	token, err := p.getToken(ctx)
	if err != nil {
		return fmt.Errorf("feishu token: %w", err)
	}
	summary := in.Object.Summary
	if summary == "" && in.Options != nil {
		summary = in.Options.Summary
	}
	if summary == "" {
		summary = in.Object.Resource + " " + in.Object.Action
	}
	baseURL := p.cfg.GatewayBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	approveURL := fmt.Sprintf("%s/cheq/approve?id=%s&approved=true", strings.TrimSuffix(baseURL, "/"), in.Object.ID)
	rejectURL := fmt.Sprintf("%s/cheq/approve?id=%s&approved=false", strings.TrimSuffix(baseURL, "/"), in.Object.ID)
	body := fmt.Sprintf("待确认请求\nTraceID: %s\nID: %s\n摘要: %s\n\n批准（点击或复制链接）: %s\n拒绝: %s",
		in.Object.TraceID, in.Object.ID, summary, approveURL, rejectURL)

	// 与 main.go 一致：默认按 user_id 发（避免 open_id cross app）；仅当以 ou_ 开头时用 open_id
	receiveIDType := "user_id"
	receiveID := p.cfg.ApprovalUserID
	if receiveID != "" && strings.HasPrefix(receiveID, "ou_") {
		receiveIDType = "open_id"
	}
	if receiveID == "" && p.cfg.ChatID != "" {
		receiveIDType = "chat_id"
		receiveID = p.cfg.ChatID
	}
	if receiveID == "" && in.Options != nil && len(in.Options.ConfirmerIDs) > 0 {
		receiveID = in.Options.ConfirmerIDs[0]
		if strings.HasPrefix(receiveID, "ou_") {
			receiveIDType = "open_id"
		} else {
			receiveIDType = "user_id"
		}
	}
	if p.cfg.ReceiveIDType != "" {
		receiveIDType = p.cfg.ReceiveIDType
	}
	if receiveID == "" {
		err := fmt.Errorf("feishu: no receive_id (approval_user_id, chat_id, or confirmer_ids)")
		fmt.Fprintf(os.Stderr, "[diting] 飞书投递: %v\n", err)
		return err
	}

	maxAttempts := p.cfg.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	initialBackoff := p.cfg.RetryInitialBackoffSeconds
	if initialBackoff <= 0 {
		initialBackoff = 1
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if p.cfg.UseCardDelivery {
			err = p.sendCard(ctx, token, receiveIDType, receiveID, in.Object.TraceID, in.Object.ID, summary, approveURL, rejectURL)
		} else {
			err = p.sendMessage(ctx, token, receiveIDType, receiveID, body)
		}
		if err == nil {
			break
		}
		if attempt < maxAttempts-1 {
			backoff := time.Duration(initialBackoff<<uint(attempt)) * time.Second
			fmt.Fprintf(os.Stderr, "[diting] 飞书投递重试 %d/%d after %v: %v\n", attempt+1, maxAttempts, backoff, err)
			time.Sleep(backoff)
		}
	}
	if err != nil && strings.Contains(err.Error(), "open_id cross app") && p.cfg.ChatID != "" {
		fmt.Fprintf(os.Stderr, "[diting] 飞书投递: open_id cross app，回退到 chat_id\n")
		if p.cfg.UseCardDelivery {
			err = p.sendCard(ctx, token, "chat_id", p.cfg.ChatID, in.Object.TraceID, in.Object.ID, summary, approveURL, rejectURL)
		} else {
			err = p.sendMessage(ctx, token, "chat_id", p.cfg.ChatID, body)
		}
	}
	return err
}

// sendCard 发送交互卡片（批准/拒绝按钮），按钮 value 为 {"request_id":"<cheq_id>","action":"approve"|"reject"}，供长连接或 HTTP 回调解析。
func (p *Provider) sendCard(ctx context.Context, token, receiveIDType, receiveID, traceID, cheqID, summary, approveURL, rejectURL string) error {
	bodyMD := fmt.Sprintf("**待确认请求**\n\nTraceID: `%s`\nID: `%s`\n摘要: %s\n\n可点击下方按钮审批，或使用链接：\n[批准](%s) | [拒绝](%s)",
		traceID, cheqID, summary, approveURL, rejectURL)
	approveVal := map[string]string{"request_id": cheqID, "action": "approve"}
	rejectVal := map[string]string{"request_id": cheqID, "action": "reject"}
	// 不使用回调 URL。卡片点击只通过长连接（card.action.trigger）回传，不填 request_url。
	configCard := map[string]interface{}{"wide_screen_mode": true}
	card := map[string]interface{}{
		"config": configCard,
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "Diting 待确认",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": bodyMD,
				},
			},
			map[string]interface{}{
				"tag": "action",
				"actions": []interface{}{
					map[string]interface{}{
						"tag":  "button",
						"text": map[string]interface{}{"tag": "plain_text", "content": "批准"},
						"type": "primary",
						"value": approveVal,
					},
					map[string]interface{}{
						"tag":  "button",
						"text": map[string]interface{}{"tag": "plain_text", "content": "拒绝"},
						"type": "default",
						"value": rejectVal,
					},
				},
			},
		},
	}
	contentBytes, _ := json.Marshal(card)
	reqBody := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "interactive",
		"content":    string(contentBytes),
	}
	payload, _ := json.Marshal(reqBody)
	url := messageAPI + "?receive_id_type=" + receiveIDType
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu message api HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(bodyBytes, &result)
	if result.Code != 0 {
		return fmt.Errorf("feishu API code=%d msg=%s", result.Code, result.Msg)
	}
	return nil
}

func (p *Provider) sendMessage(ctx context.Context, token, receiveIDType, receiveID, body string) error {
	// 飞书要求 content 为 JSON 字符串，即对 {"text":"..."} 再序列化一次
	contentJSON, _ := json.Marshal(map[string]string{"text": body})
	reqBody := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}
	payload, _ := json.Marshal(reqBody)
	url := messageAPI + "?receive_id_type=" + receiveIDType
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu message api HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(bodyBytes, &result)
	if result.Code != 0 {
		return fmt.Errorf("feishu API code=%d msg=%s", result.Code, result.Msg)
	}
	return nil
}

func (p *Provider) getToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiry) {
		t := p.token
		p.mu.RUnlock()
		return t, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Now().Before(p.expiry) {
		return p.token, nil
	}
	body := map[string]string{"app_id": p.cfg.AppID, "app_secret": p.cfg.AppSecret}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenAPI, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[diting] 飞书 token 请求失败: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	var res struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		fmt.Fprintf(os.Stderr, "[diting] 飞书 token 响应解析失败: %v\n", err)
		return "", err
	}
	if res.Code != 0 {
		err := fmt.Errorf("feishu token: code=%d msg=%s", res.Code, res.Msg)
		fmt.Fprintf(os.Stderr, "[diting] 飞书 token 失败: %v\n", err)
		return "", err
	}
	p.token = res.TenantAccessToken
	p.expiry = time.Now().Add(time.Duration(res.Expire-60) * time.Second)
	return p.token, nil
}

// 编译期保证 *Provider 实现 delivery.Provider。
var _ delivery.Provider = (*Provider)(nil)
