// Package config 提供统一配置模型与加载（YAML + env override）。
package config

// Config 根配置；敏感项由 env 覆盖（见 Load）。All-in-One 与 main/main_feishu 等入口共用。
type Config struct {
	Proxy    ProxyConfig    `yaml:"proxy"`
	Policy   PolicyConfig  `yaml:"policy"`
	CHEQ     CHEQConfig    `yaml:"cheq"`
	Delivery DeliveryConfig `yaml:"delivery"`
	Audit    AuditConfig   `yaml:"audit"`
	Ownership OwnershipConfig `yaml:"ownership"`
	Chain    ChainConfig   `yaml:"chain,omitempty"` // 私有链与 DID/存证（I-017）
	// 以下供 main_feishu / main 等入口使用（YAML 可选段）
	LLM  *LLMConfig  `yaml:"llm,omitempty"`
	Risk *RiskConfig `yaml:"risk,omitempty"`
}

// ChainConfig 链子模块配置（I-016 §7）。Enabled 为 true 时挂载 /chain/*。
type ChainConfig struct {
	Enabled                bool   `yaml:"enabled"`
	StoragePath            string `yaml:"storage_path"`              // 持久化目录（dids/batches/proofs）；空则仅内存
	MerkleBatchSize        int    `yaml:"merkle_batch_size"`          // 可选，批次大小提示；0 表示默认
	AuditBatchEnabled      bool   `yaml:"audit_batch_enabled"`        // 审计 Trace 落库后按批上链（Story 10.6）
	AuditBatchSize         int    `yaml:"audit_batch_size"`           // 每 N 条提交一批；0 表示默认 50
	AuditBatchIntervalSec  int    `yaml:"audit_batch_interval_sec"`   // 定时提交间隔（秒）；0 表示默认 30
}

// LLMConfig 大模型配置（main_feishu 等用）。
type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	BaseURL     string  `yaml:"base_url"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// RiskConfig 风险规则（main_feishu 等用）。
type RiskConfig struct {
	DangerousMethods   []string `yaml:"dangerous_methods"`
	DangerousPaths     []string `yaml:"dangerous_paths"`
	AutoApproveMethods []string `yaml:"auto_approve_methods"`
	SafeDomains        []string `yaml:"safe_domains"`
}

// ProxyConfig 代理监听与上游；L0 身份校验（MVP API Key）。
type ProxyConfig struct {
	ListenAddr     string   `yaml:"listen_addr"`      // 如 :8080
	Upstream       string   `yaml:"upstream"`         // 上游 base URL
	AllowedAPIKeys []string `yaml:"allowed_api_keys"` // 允许的 L0 API Key 列表；空表示不强制 L0 校验
}

// PolicyConfig 策略引擎配置（规则路径、热加载等）。
type PolicyConfig struct {
	RulesPath string `yaml:"rules_path"`
}

// CHEQConfig CHEQ 超时与持久化路径。
type CHEQConfig struct {
	TimeoutSeconds            int    `yaml:"timeout_seconds"`
	ReminderSecondsBeforeTimeout int `yaml:"reminder_seconds_before_timeout"` // 超时前多少秒发飞书提醒；0 表示默认 60
	PersistencePath           string `yaml:"persistence_path"`
}

// DeliveryConfig 投递配置；敏感项从 env 覆盖（DITING_FEISHU_APP_SECRET 等）。
type DeliveryConfig struct {
	Feishu FeishuConfig `yaml:"feishu"`
}

// FeishuConfig 飞书应用配置；敏感项从 env 覆盖。
type FeishuConfig struct {
	AppID                  string `yaml:"app_id"`
	AppSecret              string `yaml:"app_secret"` // 实际从 DITING_FEISHU_APP_SECRET 覆盖
	Enabled                bool   `yaml:"enabled"`
	ApprovalUserID         string `yaml:"approval_user_id"`          // 审批人 ID（见 ReceiveIDType）
	ReceiveIDType          string `yaml:"receive_id_type"`            // open_id（默认）或 user_id，避免 open_id cross app
	ApprovalTimeoutMinutes int    `yaml:"approval_timeout_minutes"` // 审批超时（分钟）
	UseMessageReply        bool   `yaml:"use_message_reply"`
	PollIntervalSeconds    int    `yaml:"poll_interval_seconds"`
	ChatID                 string `yaml:"chat_id"`       // 群聊 ID（兜底投递）
	GatewayBaseURL         string `yaml:"gateway_base_url"` // 审批链接前缀，如 http://localhost:8080，用于飞书消息中的链接
	// 长连接 + 卡片：两种方式都支持。UseCardDelivery 为 true 时发交互卡片（批准/拒绝按钮）；UseLongConnection 为 true 时启动 WebSocket 接收事件（含卡片点击）。
	UseCardDelivery   bool `yaml:"use_card_delivery"`   // 发审批为交互卡片（否则为文本+链接）
	UseLongConnection bool `yaml:"use_long_connection"` // 使用长连接接收事件（含卡片交互），需在飞书后台选「长连接」订阅
	// 飞书发送失败时的重试与退避（P1）
	RetryMaxAttempts         int `yaml:"retry_max_attempts"`          // 最大重试次数；0 表示默认 3
	RetryInitialBackoffSeconds int `yaml:"retry_initial_backoff_seconds"` // 首次退避秒数，之后指数增加；0 表示默认 1
}

// AuditConfig 审计写入路径与脱敏配置。
type AuditConfig struct {
	Path    string   `yaml:"path"`
	Redact  []string `yaml:"redact,omitempty"` // 需脱敏字段名
}

// OwnershipConfig 归属解析（静态配置或规则路径）。
type OwnershipConfig struct {
	StaticMap map[string][]string `yaml:"static_map,omitempty"` // resource -> confirmer_ids
}
