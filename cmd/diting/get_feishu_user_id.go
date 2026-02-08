// 独立运行：收到飞书消息时打印发送者的 user_id，用于填入 config.json 的 approval_user_id
// 用法：go run get_feishu_user_id.go
// 飞书开放平台 → 事件订阅 → 请求地址填 http://你的公网地址/feishu/event（可用 ngrok 等）
// 然后给应用发一条消息，终端会打印 user_id
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/feishu/event", func(w http.ResponseWriter, r *http.Request) {
		var event map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if challenge, ok := event["challenge"].(string); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"challenge": challenge})
			fmt.Println("[OK] 飞书 URL 验证已通过")
			return
		}
		header, _ := event["header"].(map[string]interface{})
		eventType, _ := header["event_type"].(string)
		if eventType != "im.message.receive_v1" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		ev, _ := event["event"].(map[string]interface{})
		sender, _ := ev["sender"].(map[string]interface{})
		senderID, _ := sender["sender_id"].(map[string]interface{})
		userID, _ := senderID["user_id"].(string)
		openID, _ := senderID["open_id"].(string)
		fmt.Println("----------------------------------------")
		fmt.Println("  收到飞书消息，发送者 ID 如下（本应用下）：")
		fmt.Println("  user_id（用于 approval_user_id）:", userID)
		fmt.Println("  open_id:", openID)
		fmt.Println("  请将 user_id 填入 config.json 的 feishu.approval_user_id")
		fmt.Println("----------------------------------------")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	fmt.Println("监听 http://0.0.0.0:9000/feishu/event")
	fmt.Println("飞书事件订阅 URL 填: http://你的公网地址/feishu/event")
	fmt.Println("给应用发一条消息后，此处会打印 user_id")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
