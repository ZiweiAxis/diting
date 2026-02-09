// 独立运行：启动 HTTP 服务，通过飞书「事件订阅 - HTTP 回调」接收用户发消息事件，打印该用户的 open_id 与 user_id。
// 说明：飞书对接接收事件有两种方式——长连接（WebSocket，推荐）与 HTTP 回调。本工具为 HTTP 回调方式；
// 若你使用长连接，可在长连接建立后从收消息事件中直接获取 user_id/open_id，无需本监听器。
// 用法：cd cmd/diting && go run feishu_id_listener.go
// 需在飞书开放平台配置事件订阅（选择 HTTP 回调）并填请求地址（可用 ngrok 暴露 9000），订阅 im.message.receive_v1。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	port := "9000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	addr := ":" + port

	http.HandleFunc("/feishu/event", handleFeishuEvent)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Feishu ID Listener. Use POST /feishu/event as Feishu event subscription URL.\n"))
	})

	fmt.Println("========================================")
	fmt.Println("  飞书 open_id / user_id 监听服务")
	fmt.Println("========================================")
	fmt.Printf("  本机: http://0.0.0.0%s/feishu/event\n", addr)
	fmt.Println("  飞书开放平台 → 事件订阅 → 选择 HTTP 回调 → 请求地址填: http://<你的公网地址>/feishu/event")
	fmt.Println("  本地需用 ngrok 等暴露端口:", port)
	fmt.Println("  配置好后，给应用发一条消息，此处会打印发送者的 open_id 与 user_id。")
	fmt.Println("========================================")
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleFeishuEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// URL 验证：飞书首次配置请求地址时会带 challenge
	if challenge, ok := event["challenge"].(string); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"challenge":"` + challenge + `"}`))
		fmt.Println("[OK] 飞书请求地址验证已通过")
		return
	}
	// 事件推送（HTTP 回调模式）
	header, _ := event["header"].(map[string]interface{})
	eventType, _ := header["event_type"].(string)
	if eventType != "im.message.receive_v1" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
		return
	}
	ev, _ := event["event"].(map[string]interface{})
	sender, _ := ev["sender"].(map[string]interface{})
	senderID, _ := sender["sender_id"].(map[string]interface{})
	userID, _ := senderID["user_id"].(string)
	openID, _ := senderID["open_id"].(string)

	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("  收到飞书消息，发送者 ID（本应用下）")
	fmt.Println("========================================")
	fmt.Printf("  open_id:  %s\n", openID)
	fmt.Printf("  user_id:  %s\n", userID)
	fmt.Println("----------------------------------------")
	fmt.Println("  建议：为避免 open_id cross app(99992361)，请使用 user_id：")
	fmt.Println("  1. config.json 或 .env 中设置 approval_user_id = 上面的 user_id")
	fmt.Println("  2. 设置 receive_id_type = user_id（或 DITING_FEISHU_RECEIVE_ID_TYPE=user_id）")
	fmt.Println("========================================")
	fmt.Println("")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
