// All-in-One 入口：加载配置、装配各组件占位实现、启动探针与代理监听。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"diting/internal/audit"
	"diting/internal/chain"
	"diting/internal/cheq"
	"diting/internal/config"
	"diting/internal/delivery"
	feishudelivery "diting/internal/delivery/feishu"
	"diting/internal/ownership"
	"diting/internal/policy"
	"diting/internal/proxy"
	chainpkg "diting/pkg/chain"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml (or set CONFIG_PATH)")
	validateOnly := flag.Bool("validate", false, "load config (and policy rules if set), then exit 0 on success or 1 on error (Epic 4.1)")
	flag.Parse()

	// 工作目录：watch/air 下 cwd 可能不是 cmd/diting，用可执行文件所在目录的上级作为配置根
	workDir := "."
	if execPath, err := os.Executable(); err == nil {
		parent := filepath.Clean(filepath.Join(filepath.Dir(execPath), ".."))
		if _, err := os.Stat(filepath.Join(parent, ".env")); err == nil {
			workDir = parent
			_ = os.Chdir(workDir)
			fmt.Fprintf(os.Stderr, "[diting] 工作目录: %s（由可执行文件位置推断，watch 下可正确加载 .env）\n", workDir)
		}
	}
	// 配置：.env 覆盖敏感项；主配置仅 YAML（路径已相对于 workDir）
	_ = config.LoadEnvFile(".env", true)
	if *configPath == "" {
		*configPath = os.Getenv("CONFIG_PATH")
	}
	if *configPath == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			*configPath = "config.yaml"
		} else {
			*configPath = "config.example.yaml"
		}
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}
	if *validateOnly {
		// 校验策略规则文件（若配置了）
		if cfg.Policy.RulesPath != "" {
			if _, err := policy.NewEngineImpl(cfg.Policy.RulesPath); err != nil {
				fmt.Fprintf(os.Stderr, "policy rules validate: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "[diting] config validate ok: %s\n", *configPath)
		os.Exit(0)
	}

	// 本地调试：配置来源与飞书投递诊断（不打印敏感值）
	if _, err := os.Stat(".env"); err == nil {
		fmt.Fprintf(os.Stderr, "[diting] 配置: .env 已加载\n")
	} else {
		fmt.Fprintf(os.Stderr, "[diting] 配置: .env 未找到（本地调试可复制 .env.example 为 .env 并填写 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID 或 DITING_FEISHU_CHAT_ID）\n")
	}
	fmt.Fprintf(os.Stderr, "[diting] 配置: YAML=%s\n", *configPath)
	if cfg.Delivery.Feishu.Enabled {
		hasApp := cfg.Delivery.Feishu.AppID != "" && cfg.Delivery.Feishu.AppSecret != ""
		hasTarget := cfg.Delivery.Feishu.ApprovalUserID != "" || cfg.Delivery.Feishu.ChatID != ""
		fmt.Fprintf(os.Stderr, "[diting] 飞书: app_id/app_secret=%v, approval_user_id或chat_id=%v\n", hasApp, hasTarget)
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
		if len(cfg.Delivery.Feishu.ApprovalUserIDs) > 0 || cfg.Delivery.Feishu.ApprovalUserID != "" || cfg.Delivery.Feishu.ChatID != "" {
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
	// I-008: 支持多审批人默认列表；无 static_map 时也用 defaultIDs
	defaultApprovalIDs := cfg.Delivery.Feishu.ApprovalUserIDs
	ownershipResolver = ownership.NewStaticResolver(cfg.Ownership.StaticMap, defaultApprovalIDs)
	if cfg.CHEQ.PersistencePath != "" {
		store, err := cheq.NewJSONStore(cfg.CHEQ.PersistencePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cheq store: %v\n", err)
			os.Exit(1)
		}
		cheqEngine = cheq.NewEngineImpl(store, cfg.CHEQ.TimeoutSeconds, ownershipResolver, deliveryProvider, cfg.Delivery.Feishu.ApprovalPolicy)
	} else {
		cheqEngine = cheq.NewStubEngine()
	}
	var auditStore audit.Store
	if cfg.Audit.Path != "" {
		as, err := audit.NewJSONLStore(cfg.Audit.Path, cfg.Audit.Redact)
		if err != nil {
			fmt.Fprintf(os.Stderr, "audit store: %v\n", err)
			os.Exit(1)
		}
		auditStore = as
	} else {
		auditStore = audit.NewStubStore()
	}
	var ledger chainpkg.Ledger
	var chainSrv *chain.Server
	if cfg.Chain.Enabled {
		store := chainpkg.NewLocalStoreWithPath(cfg.Chain.StoragePath)
		ledger = chainpkg.NewLedger(store)
		chainSrv = chain.NewServer(ledger)
		if cfg.Chain.AuditBatchEnabled {
			batchSize := cfg.Chain.AuditBatchSize
			if batchSize <= 0 {
				batchSize = 50
			}
			intervalSec := cfg.Chain.AuditBatchIntervalSec
			if intervalSec <= 0 {
				intervalSec = 30
			}
			bridge := chain.NewAuditChainBridge(auditStore, ledger, batchSize, time.Duration(intervalSec)*time.Second)
			bridge.Start()
			defer bridge.Stop()
			auditStore = bridge
			fmt.Fprintf(os.Stderr, "[diting] 审计存证上链已启用（batch_size=%d, interval=%ds）\n", batchSize, intervalSec)
		}
	}
	reviewRequiresApproval := cfg.CHEQ.PersistencePath != "" // 使用持久化 CHEQ 时轮询等待确认
	var approvalMatcher *ownership.RuleMatcher
	if len(cfg.CHEQ.ApprovalRules) > 0 {
		rules := make([]struct {
			PathPrefix      string
			RiskLevel       string
			TimeoutSeconds  int
			ApprovalUserIDs []string
			ApprovalPolicy  string
		}, len(cfg.CHEQ.ApprovalRules))
		for i, r := range cfg.CHEQ.ApprovalRules {
			rules[i] = struct {
				PathPrefix      string
				RiskLevel       string
				TimeoutSeconds  int
				ApprovalUserIDs []string
				ApprovalPolicy  string
			}{
				PathPrefix:      r.PathPrefix,
				RiskLevel:       r.RiskLevel,
				TimeoutSeconds:  r.TimeoutSeconds,
				ApprovalUserIDs: append([]string(nil), r.ApprovalUserIDs...),
				ApprovalPolicy:  r.ApprovalPolicy,
			}
		}
		defTimeout := cfg.CHEQ.TimeoutSeconds
		if defTimeout <= 0 {
			defTimeout = 300
		}
		defPolicy := cfg.Delivery.Feishu.ApprovalPolicy
		if defPolicy != "all" {
			defPolicy = "any"
		}
		approvalMatcher = ownership.NewRuleMatcher(rules, ownership.ApprovalRuleMatch{
			TimeoutSeconds:  defTimeout,
			ApprovalUserIDs: append([]string(nil), cfg.Delivery.Feishu.ApprovalUserIDs...),
			ApprovalPolicy:  defPolicy,
		})
	}
	srv := proxy.NewServer(cfg, policyEngine, cheqEngine, deliveryProvider, auditStore, ownershipResolver, reviewRequiresApproval, approvalMatcher)
	if cfg.Chain.Enabled {
		srv.SetChainHandler(chainSrv.Handler())
		fmt.Fprintf(os.Stderr, "[diting] 链子模块已启用，/chain/did/*、/chain/audit/*、/chain/health 可用\n")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// SIGHUP 触发热加载策略规则（Story 2.4）
	if pe, ok := policyEngine.(*policy.EngineImpl); ok {
		sigReload := make(chan os.Signal, 1)
		signal.Notify(sigReload, syscall.SIGHUP)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-sigReload:
					if err := pe.Reload(); err != nil {
						log.Printf("[diting] policy reload failed: %v", err)
					} else {
						log.Printf("[diting] policy rules reloaded successfully")
					}
				}
			}
		}()
		fmt.Fprintf(os.Stderr, "[diting] SIGHUP will reload policy rules\n")
	}
	if cfg.Delivery.Feishu.Enabled && cfg.Delivery.Feishu.UseLongConnection {
		feishudelivery.RunLongConnection(ctx, cfg.Delivery.Feishu, func(cheqID string, approved bool) error {
			return cheqEngine.Submit(context.Background(), cheqID, approved, "")
		})
		fmt.Fprintf(os.Stderr, "[diting] 飞书长连接已启动（卡片交互事件将在此处理）\n")
	}
	if err := srv.Serve(ctx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}
