// Package feishu 长连接：使用官方 SDK 建立 WebSocket，接收事件（含卡片按钮点击），并回调 onCardAction 完成审批。
package feishu

import (
	"context"
	"fmt"
	"os"
	"time"

	"diting/internal/config"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// RunLongConnection 在后台建立飞书长连接，接收 EVENT_CALLBACK；若为卡片交互（action.value.request_id + action），则调用 onCardAction。
// 需在飞书开放平台选择「使用长连接接收事件」并订阅相应事件。ctx 取消时退出。
func RunLongConnection(ctx context.Context, cfg config.FeishuConfig, onCardAction func(cheqID string, approved bool) error) {
	if !cfg.Enabled || cfg.AppID == "" || cfg.AppSecret == "" {
		return
	}
	go runWSLoop(ctx, cfg, onCardAction)
}

func runWSLoop(ctx context.Context, cfg config.FeishuConfig, onCardAction func(cheqID string, approved bool) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		eventHandler := dispatcher.NewEventDispatcher("", "").
			OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
				if event == nil || event.Event == nil || event.Event.Action == nil {
					return &callback.CardActionTriggerResponse{}, nil
				}
				return handleWSCardAction(event.Event.Action.Value, onCardAction), nil
			})
		client := larkws.NewClient(cfg.AppID, cfg.AppSecret, larkws.WithEventHandler(eventHandler))
		fmt.Fprintf(os.Stderr, "[diting] 飞书长连接已建立，等待卡片交互事件...\n")
		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := client.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "[diting] 飞书长连接错误: %v\n", err)
			}
		}()
		select {
		case <-ctx.Done():
			return
		case <-done:
			// 连接断开，重连
		}
		time.Sleep(5 * time.Second)
	}
}

// handleWSCardAction 处理 SDK 事件里的卡片点击（event_type=card.action.trigger），不走 HTTP 回调。
func handleWSCardAction(value map[string]interface{}, onCardAction func(cheqID string, approved bool) error) *callback.CardActionTriggerResponse {
	if value == nil {
		return &callback.CardActionTriggerResponse{}
	}
	requestID, _ := value["request_id"].(string)
	actionStr, _ := value["action"].(string)
	if requestID == "" || actionStr == "" {
		return &callback.CardActionTriggerResponse{}
	}
	approved := actionStr == "approve"
	if err := onCardAction(requestID, approved); err != nil {
		fmt.Fprintf(os.Stderr, "[diting] 飞书卡片审批 Submit: %v\n", err)
		return &callback.CardActionTriggerResponse{}
	}
	fmt.Fprintf(os.Stderr, "[diting] 飞书长连接卡片审批: id=%s approved=%v (event_type=%s)\n", requestID, approved, "card.action.trigger")
	return &callback.CardActionTriggerResponse{}
}
