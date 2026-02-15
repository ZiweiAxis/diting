package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// configJSON 与旧版 JSON 配置结构兼容，用于 LoadEnvFromConfigJSON（可选，主流程已用 config.yaml + .env）。
type configJSON struct {
	Proxy  *struct{ Listen string `json:"listen"` } `json:"proxy"`
	Feishu *struct {
		Enabled        bool   `json:"enabled"`
		AppID          string `json:"app_id"`
		AppSecret      string `json:"app_secret"`
		ApprovalUserID string `json:"approval_user_id"`
		ChatID         string `json:"chat_id"`
	} `json:"feishu"`
}

// LoadEnvFile 从 path 读取 .env 风格文件（KEY=VALUE），并 set 到当前进程环境变量。
// 空行与 # 开头行忽略；不覆盖已存在的环境变量（可选：传 true 则覆盖）。
// 在 Load 之前调用，则 YAML 的 env 覆盖会使用 .env 中的值。
func LoadEnvFile(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
			val = strings.Trim(val, `"`)
		}
		if key == "" {
			continue
		}
		if override || os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return sc.Err()
}

// LoadEnvFromConfigJSON 从 path 读取旧版 JSON 配置文件，
// 将其中的 feishu、proxy 等写入环境变量（仅当该 env 尚未设置时）。主流程已用 config.yaml + .env，此函数为可选兼容。
func LoadEnvFromConfigJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var c configJSON
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	setEnvIfEmpty := func(key, val string) {
		if val != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	if c.Feishu != nil && c.Feishu.Enabled {
		setEnvIfEmpty("DITING_FEISHU_APP_ID", c.Feishu.AppID)
		setEnvIfEmpty("DITING_FEISHU_APP_SECRET", c.Feishu.AppSecret)
		setEnvIfEmpty("DITING_FEISHU_APPROVAL_USER_ID", c.Feishu.ApprovalUserID)
		setEnvIfEmpty("DITING_FEISHU_CHAT_ID", c.Feishu.ChatID)
	}
	// 不注入 proxy.listen，避免覆盖 YAML 的 listen_addr（如 :8080），
	// 否则网关会占 8081 与上游端口冲突；若需用 8081 可在 .env 设 DITING_PROXY_LISTEN。
	return nil
}

// Load 从 path 加载 YAML 配置；敏感项由环境变量覆盖。
// 支持 DITING_FEISHU_APP_SECRET、DITING_FEISHU_APP_ID 等覆盖 delivery.feishu。
// CONFIG_PATH 或 -config 由调用方传入 path 即可。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config load: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}
	applyEnvOverrides(&c)
	// 多审批人兼容：无 approval_user_ids 时用 approval_user_id 作为单元素
	if len(c.Delivery.Feishu.ApprovalUserIDs) == 0 && c.Delivery.Feishu.ApprovalUserID != "" {
		c.Delivery.Feishu.ApprovalUserIDs = []string{c.Delivery.Feishu.ApprovalUserID}
	}
	if c.Delivery.Feishu.ApprovalPolicy != "all" {
		c.Delivery.Feishu.ApprovalPolicy = "any"
	}
	return &c, nil
}

// applyEnvOverrides 用 DITING_ 前缀环境变量覆盖敏感或常用项。
func applyEnvOverrides(c *Config) {
	if v := os.Getenv("DITING_FEISHU_APP_ID"); v != "" {
		c.Delivery.Feishu.AppID = v
	}
	if v := os.Getenv("DITING_FEISHU_APP_SECRET"); v != "" {
		c.Delivery.Feishu.AppSecret = v
	}
	if v := os.Getenv("DITING_FEISHU_APPROVAL_USER_ID"); v != "" {
		c.Delivery.Feishu.ApprovalUserID = v
		if len(c.Delivery.Feishu.ApprovalUserIDs) == 0 {
			c.Delivery.Feishu.ApprovalUserIDs = []string{v}
		}
	}
	if v := os.Getenv("DITING_FEISHU_CHAT_ID"); v != "" {
		c.Delivery.Feishu.ChatID = v
	}
	if v := os.Getenv("DITING_FEISHU_RECEIVE_ID_TYPE"); v != "" {
		c.Delivery.Feishu.ReceiveIDType = v
	}
	if v := os.Getenv("DITING_GATEWAY_BASE_URL"); v != "" {
		c.Delivery.Feishu.GatewayBaseURL = v
	}
	if v := os.Getenv("DITING_FEISHU_USE_CARD_DELIVERY"); v != "" {
		c.Delivery.Feishu.UseCardDelivery = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("DITING_FEISHU_USE_LONG_CONNECTION"); v != "" {
		c.Delivery.Feishu.UseLongConnection = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("DITING_PROXY_LISTEN"); v != "" {
		c.Proxy.ListenAddr = v
	}
	if v := os.Getenv("DITING_CHEQ_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.CHEQ.TimeoutSeconds = n
		}
	}
	if v := os.Getenv("DITING_CHEQ_REMINDER_SECONDS_BEFORE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.CHEQ.ReminderSecondsBeforeTimeout = n
		}
	}
	if v := os.Getenv("DITING_FEISHU_RETRY_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Delivery.Feishu.RetryMaxAttempts = n
		}
	}
	if v := os.Getenv("DITING_FEISHU_RETRY_INITIAL_BACKOFF_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Delivery.Feishu.RetryInitialBackoffSeconds = n
		}
	}
	if c.LLM != nil {
		if v := os.Getenv("DITING_LLM_BASE_URL"); v != "" {
			c.LLM.BaseURL = v
		}
		if v := os.Getenv("DITING_LLM_API_KEY"); v != "" {
			c.LLM.APIKey = v
		}
		if v := os.Getenv("DITING_LLM_MODEL"); v != "" {
			c.LLM.Model = v
		}
		if v := os.Getenv("DITING_LLM_PROVIDER"); v != "" {
			c.LLM.Provider = v
		}
		if v := os.Getenv("DITING_LLM_MAX_TOKENS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.LLM.MaxTokens = n
			}
		}
		if v := os.Getenv("DITING_LLM_TEMPERATURE"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				c.LLM.Temperature = f
			}
		}
	}
}
