// Sentinel-AI DNS 劫持器
// 将 Agent 的所有 DNS 查询劫持到 Sentinel-AI WAF 网关

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/fatih/color"
)

// ==================== 配置 ====================
type Config struct {
	ListenAddr    string   `json:"listen_addr"`    // DNS 监听地址 (默认 :53)
	GatewayIP     string   `json:"gateway_ip"`     // Sentinel-AI 网关 IP
	UpstreamDNS   string   `json:"upstream_dns"`   // 上游 DNS (8.8.8.8:53)
	HijackDomains []string `json:"hijack_domains"`  // 需要劫持的域名
	UpstreamDNSs  []string `json:"upstream_dnss"`   // 上游 DNS 列表
	LogFile       string   `json:"log_file"`       // 日志文件
	CacheTTL      int      `json:"cache_ttl"`      // DNS 缓存 TTL (秒)
}

var config = Config{
	ListenAddr:    ":53",
	GatewayIP:     "10.0.0.1",
	UpstreamDNS:   "8.8.8.8:53",
	HijackDomains: []string{
		"api.example.com",
		"db.example.com",
		"auth.example.com",
	},
	UpstreamDNSs: []string{
		"8.8.8.8:53",
		"1.1.1.1:53",
		"223.5.5.5:53",
	},
	CacheTTL: 300,
}

// ==================== DNS 缓存 ====================
type DNSCache struct {
	mu     sync.RWMutex
	entries map[string]*CacheEntry
}

type CacheEntry struct {
	Answer   []dns.RR
	ExpireAt time.Time
}

func NewDNSCache() *DNSCache {
	return &DNSCache{
		entries: make(map[string]*CacheEntry),
	}
}

func (c *DNSCache) Get(key string) ([]dns.RR, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpireAt) {
		delete(c.entries, key)
		return nil, false
	}

	return entry.Answer, true
}

func (c *DNSCache) Set(key string, answer []dns.RR, ttl uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Answer:   answer,
		ExpireAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
}

// ==================== DNS 劫持器 ====================
type DNSHijacker struct {
	config    Config
	cache     *DNSCache
	dnsClient *dns.Client
	patterns  []string
	mu        sync.RWMutex
}

func NewDNSHijacker(config Config) *DNSHijacker {
	h := &DNSHijacker{
		config:    config,
		cache:     NewDNSCache(),
		dnsClient: &dns.Client{Timeout: 2 * time.Second},
		patterns:  make([]string, 0),
	}

	// 编译劫持域名模式
	for _, domain := range config.HijackDomains {
		h.addPattern(domain)
	}

	return h
}

// 添加劫持域名模式
func (h *DNSHijacker) addPattern(domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	if strings.Contains(domain, "*.") {
		// 通配符域名
		h.patterns = append(h.patterns, domain)
	} else {
		h.patterns = append(h.patterns, domain)
	}
}

// 判断是否需要劫持
func (h *DNSHijacker) shouldHijack(domain string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// 精确匹配
	for _, pattern := range h.patterns {
		if pattern == domain {
			return true
		}
	}

	// 通配符匹配
	for _, pattern := range h.patterns {
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if strings.HasSuffix(domain, suffix) {
				return true
			}
		}
	}

	return false
}

// 创建劫持响应 (返回网关 IP)
func (h *DNSHijacker) createHijackResponse(domain string) []dns.RR {
	rr, err := dns.NewRR(fmt.Sprintf("%s 300 IN A %s", domain, h.config.GatewayIP))
	if err != nil {
		return nil
	}
	return []dns.RR{rr}
}

// 转发到上游 DNS
func (h *DNSHijacker) forwardDNS(q dns.Question) ([]dns.RR, error) {
	m := new(dns.Msg)
	m.SetQuestion(q.Name, q.Qtype)
	m.RecursionDesired = true

	// 尝试所有上游 DNS
	for _, upstream := range h.config.UpstreamDNSs {
		r, _, err := h.dnsClient.Exchange(m, upstream)
		if err == nil && len(r.Answer) > 0 {
			return r.Answer, nil
		}
	}

	return nil, fmt.Errorf("all upstream DNS servers failed")
}

// 处理 DNS 请求
func (h *DNSHijacker) HandleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	m.Authoritative = true

	for _, q := range r.Question {
		domain := strings.ToLower(strings.TrimSuffix(q.Name, "."))
		cacheKey := fmt.Sprintf("%s:%d", q.Name, q.Qtype)

		color.Cyan("\n[%s] DNS 查询: %s (类型: %d)", time.Now().Format("15:04:05"), q.Name, q.Qtype)

		// 检查缓存
		if answer, ok := h.cache.Get(cacheKey); ok {
			m.Answer = append(m.Answer, answer...)
			color.Green("  命中缓存")
		} else if h.shouldHijack(domain) {
			// 劫持: 返回网关 IP
			hijackAnswer := h.createHijackResponse(q.Name)
			m.Answer = append(m.Answer, hijackAnswer...)
			h.cache.Set(cacheKey, hijackAnswer, uint32(config.CacheTTL))
			color.Yellow("  🎯 劫持: 返回网关 IP (%s)", h.config.GatewayIP)
			color.Yellow("     原始域名: %s", q.Name)
		} else {
			// 转发到上游 DNS
			forwardAnswer, err := h.forwardDNS(q)
			if err != nil {
				color.Red("  转发失败: %v", err)
				m.SetRcode(r, dns.RcodeServerFailure)
				w.WriteMsg(m)
				return
			}
			m.Answer = append(m.Answer, forwardAnswer...)

			// 缓存
			if len(forwardAnswer) > 0 {
				ttl := forwardAnswer[0].Header().Ttl
				h.cache.Set(cacheKey, forwardAnswer, ttl)
				color.Green("  转发成功: %s", forwardAnswer[0])
			}
		}
	}

	w.WriteMsg(m)
}

// ==================== DNS 日志 ====================
type DNSLogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Query      string    `json:"query"`
	QueryType  uint16    `json:"query_type"`
	ClientIP   string    `json:"client_ip"`
	Hijacked   bool      `json:"hijacked"`
	GatewayIP  string    `json:"gateway_ip,omitempty"`
	Answer     string    `json:"answer,omitempty"`
	Upstream   string    `json:"upstream,omitempty"`
}

func logDNSEntry(entry DNSLogEntry) {
	// TODO: 写入日志文件
	data, _ := json.Marshal(entry)
	log.Println(string(data))
}

// ==================== 主程序 ====================
func main() {
	// 打印标题
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Sentinel-AI DNS 劫持器 v1.0                  ║")
	color.Cyan("║    将所有 Agent DNS 查询劫持到 WAF 网关               ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	color.White("")

	// 显示配置
	color.Yellow("配置:")
	fmt.Printf("  监听地址: %s\n", config.ListenAddr)
	fmt.Printf("  网关 IP:  %s\n", config.GatewayIP)
	fmt.Printf("  上游 DNS: %s\n", config.UpstreamDNS)
	fmt.Printf("  劫持域名 (%d):\n", len(config.HijackDomains))
	for _, domain := range config.HijackDomains {
		color.Red("    - %s", domain)
	}
	fmt.Println()

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 创建 DNS 劫持器
	hijacker := NewDNSHijacker(config)

	// 创建 DNS 服务器
	server := &dns.Server{
		Addr:    config.ListenAddr,
		Net:     "udp",
		Handler: hijacker,
	}

	// 启动 TCP DNS 服务器 (可选)
	go func() {
		tcpServer := &dns.Server{
			Addr:    config.ListenAddr,
			Net:     "tcp",
			Handler: hijacker,
		}
		log.Fatal(tcpServer.ListenAndServe())
	}()

	// 启动 UDP DNS 服务器
	color.Green("✓ DNS 劫持器启动成功")
	fmt.Printf("  监听地址: %s (UDP/TCP)\n", config.ListenAddr)
	fmt.Printf("  网关 IP:  %s\n", config.GatewayIP)
	fmt.Println()
	color.Yellow("所有配置的域名都将解析到网关 IP")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 测试命令
	color.White("测试命令:")
	fmt.Println("  nslookup api.example.com 10.0.0.1")
	fmt.Println("  dig @10.0.0.1 api.example.com")
	fmt.Println("  host api.example.com 10.0.0.1")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}
