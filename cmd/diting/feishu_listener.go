package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
)

// 启动飞书事件监听服务
func startFeishuListener() {
	color.Cyan("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接服务...")
	
	// 启动 HTTP 服务器用于接收事件回调
	go startFeishuHTTPServer()
	
	color.Green("✓ 飞书事件监听服务已启动")
	color.White("  监听地址: http://localhost:9000/feishu/event")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// 启动 HTTP 服务器
func startFeishuHTTPServer() {
	http.HandleFunc("/feishu/event", handleFeishuEvent)
	http.HandleFunc("/feishu/card", handleFeishuCard)
	
	log.Fatal(http.ListenAndServe(":9000", nil))
}

// 处理飞书事件
func handleFeishuEvent(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 处理 URL 验证
	if challenge, ok := event["challenge"].(string); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"challenge": challenge,
		})
		color.Green("✓ 飞书 URL 验证成功")
		return
	}

	// 处理消息事件
	if header, ok := event["header"].(map[string]interface{}); ok {
		eventType := header["event_type"].(string)
		
		if eventType == "im.message.receive_v1" {
			handleMessageReceive(event)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// 处理接收到的消息
func handleMessageReceive(event map[string]interface{}) {
	eventData, ok := event["event"].(map[string]interface{})
	if !ok {
		return
	}

	// 获取消息内容
	message, ok := eventData["message"].(map[string]interface{})
	if !ok {
		return
	}

	messageType := message["message_type"].(string)
	if messageType != "text" {
		return
	}

	// 解析文本内容
	content := message["content"].(string)
	var textContent map[string]string
	json.Unmarshal([]byte(content), &textContent)
	text := textContent["text"]

	// 获取发送者信息
	sender, ok := eventData["sender"].(map[string]interface{})
	if !ok {
		return
	}

	senderID := sender["sender_id"].(map[string]interface{})
	userID := senderID["user_id"].(string)
	openID := senderID["open_id"].(string)

	color.Cyan("\n[%s] 收到飞书消息", time.Now().Format("15:04:05"))
	fmt.Printf("  发送者: %s (open_id: %s)\n", userID, openID)
	fmt.Printf("  内容: %s\n", text)

	// 检查是否是审批回复
	checkApprovalReply(text, userID, openID)
}

// 检查审批回复
func checkApprovalReply(text, userID, openID string) {
	text = strings.ToLower(strings.TrimSpace(text))
	
	// 批准关键词
	approveKeywords := []string{"批准", "approve", "y", "yes", "同意"}
	// 拒绝关键词
	rejectKeywords := []string{"拒绝", "reject", "n", "no", "不同意"}

	var decision string
	for _, keyword := range approveKeywords {
		if text == keyword {
			decision = "approved"
			break
		}
	}
	
	if decision == "" {
		for _, keyword := range rejectKeywords {
			if text == keyword {
				decision = "rejected"
				break
			}
		}
	}

	if decision != "" {
		// 更新审批请求状态
		approvalRequests.Range(func(key, value interface{}) bool {
			req := value.(*ApprovalRequest)
			if req.Status == "pending" {
				req.Status = decision
				approvalRequests.Store(key, req)
				
				color.Green("  ✓ 审批决策: %s", decision)
				
				// 发送确认消息
				confirmMsg := "✅ 已批准操作"
				if decision == "rejected" {
					confirmMsg = "❌ 已拒绝操作"
				}
				sendFeishuMessage(openID, confirmMsg)
				
				return false // 停止遍历
			}
			return true
		})
	}
}

// 处理卡片回调
func handleFeishuCard(w http.ResponseWriter, r *http.Request) {
	var callback map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 处理卡片按钮点击
	action, ok := callback["action"].(map[string]interface{})
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	value, ok := action["value"].(map[string]interface{})
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	requestID := value["request_id"].(string)
	actionType := value["action"].(string) // "approve" or "reject"

	// 更新审批状态
	if val, ok := approvalRequests.Load(requestID); ok {
		req := val.(*ApprovalRequest)
		if actionType == "approve" {
			req.Status = "approved"
		} else {
			req.Status = "rejected"
		}
		approvalRequests.Store(requestID, req)
		
		color.Green("  ✓ 卡片审批决策: %s", actionType)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
