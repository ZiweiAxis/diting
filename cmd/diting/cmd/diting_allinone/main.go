// All-in-One 入口：加载配置、装配各组件占位实现、启动探针与代理监听。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/config"
	"diting/internal/delivery"
	feishudelivery "diting/internal/delivery/feishu"
	"diting/internal/ownership"
	"diting/internal/policy"
	"diting/internal/proxy"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml (or set CONFIG_PATH)")
	flag.Parse()
	// 配置优先级：.env（覆盖）> config.json（已有飞书等配置）> YAML 内默认
	_ = config.LoadEnvFile(".env", true)
	_ = config.LoadEnvFromConfigJSON("config.json")
	if *configPath == "" {
		*configPath = os.Getenv("CONFIG_PATH")
	}
	if *configPath == "" {
		*configPath = "config.example.yaml"
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	// 策略：有规则路径则用内置引擎，否则占位恒放行
	var policyEngine policy.Engine
	if cfg.Policy.RulesPath != "" {
		pe, err := policy.NewEngineImpl(cfg.Policy.RulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "policy engine: %v\n", err)
			os.Exit(1)
		}
		policyEngine = pe
	} else {
		policyEngine = &policy.StubEngine{}
	}
	var cheqEngine cheq.Engine
	var deliveryProvider delivery.Provider
	if cfg.Delivery.Feishu.Enabled && cfg.Delivery.Feishu.AppID != "" && cfg.Delivery.Feishu.AppSecret != "" {
		deliveryProvider = feishudelivery.NewProvider(cfg.Delivery.Feishu)
		if cfg.Delivery.Feishu.ApprovalUserID != "" || cfg.Delivery.Feishu.ChatID != "" {
			fmt.Fprintf(os.Stderr, "[diting] 飞书投递已启用，审批人将收到待确认消息\n")
		} else {
			fmt.Fprintf(os.Stderr, "[diting] 飞书投递已启用，但未配置 approval_user_id/chat_id，请设置 DITING_FEISHU_APPROVAL_USER_ID 或 static_map 以收到消息\n")
		}
	} else {
		deliveryProvider = &delivery.StubProvider{}
		if cfg.Delivery.Feishu.Enabled {
			fmt.Fprintf(os.Stderr, "[diting] 飞书未配置 app_id/app_secret，使用占位投递（不发飞书）。设置 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET 后可见飞书审批流程\n")
		}
	}
	var ownershipResolver ownership.Resolver
	if len(cfg.Ownership.StaticMap) > 0 {
		ownershipResolver = ownership.NewStaticResolver(cfg.Ownership.StaticMap)
	} else {
		ownershipResolver = &ownership.StubResolver{}
	}
	if cfg.CHEQ.PersistencePath != "" {
		store, err := cheq.NewJSONStore(cfg.CHEQ.PersistencePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cheq store: %v\n", err)
			os.Exit(1)
		}
		cheqEngine = cheq.NewEngineImpl(store, cfg.CHEQ.TimeoutSeconds, ownershipResolver, deliveryProvider)
	} else {
		cheqEngine = cheq.NewStubEngine()
	}
	var auditStore audit.Store
	if cfg.Audit.Path != "" {
		as, err := audit.NewJSONLStore(cfg.Audit.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "audit store: %v\n", err)
			os.Exit(1)
		}
		auditStore = as
	} else {
		auditStore = audit.NewStubStore()
	}
	reviewRequiresApproval := cfg.CHEQ.PersistencePath != "" // 使用持久化 CHEQ 时轮询等待确认
	srv := proxy.NewServer(cfg, policyEngine, cheqEngine, deliveryProvider, auditStore, ownershipResolver, reviewRequiresApproval)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if cfg.Delivery.Feishu.Enabled && cfg.Delivery.Feishu.UseLongConnection {
		feishudelivery.RunLongConnection(ctx, cfg.Delivery.Feishu, func(cheqID string, approved bool) error {
			return cheqEngine.Submit(context.Background(), cheqID, approved)
		})
		fmt.Fprintf(os.Stderr, "[diting] 飞书长连接已启动（卡片交互事件将在此处理）\n")
	}
	if err := srv.Serve(ctx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}
