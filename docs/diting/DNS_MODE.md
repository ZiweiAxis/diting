# Diting 双接入：Proxy 与 DNS 模式说明

## 两种接入方式

| 方式 | 说明 | 适用 |
|------|------|------|
| **Proxy** | Agent 配置 HTTP(S) 代理指向 Diting（如 `http://diting-host:8080`），所有经代理的请求由 Diting 拦截、策略评估、确认后转发。 | 默认推荐；当前 All-in-One 即开即用。 |
| **DNS** | Agent 不配代理，仅将「业务域名」的 DNS 解析指向运行 Diting 的机器（或网关）；需配合 CoreDNS/自建 DNS 或 hosts 将指定域名解析到 Diting 所在 IP，且 Diting 或前置需监听 80/443。 | 希望 Agent 无感知、不改代理配置时使用。 |

## Proxy 模式（当前默认）

1. 启动 Diting All-in-One（`make run` 或容器），监听 `:8080`。
2. 将 Agent 的 HTTP 代理设为 `http://<diting-host>:8080`。
3. 验证：发请求后查看 Diting 日志或 `GET /debug/audit?trace_id=xxx`，确认请求被拦截与审计。

## DNS 模式（可选）

1. **部署 Diting**：All-in-One 或网关监听 80/443（或由 Nginx/Ingress 反代到 Diting 8080）。
2. **DNS 解析**：将需要经 Diting 的业务域名（如 `api.mycompany.com`）解析到运行 Diting（或前置网关）的机器 IP。  
   - 方式 A：在 CoreDNS 的 `hosts` 或 `file` 中为该域名配置 A 记录指向该 IP；客户端 DNS 指向该 CoreDNS。  
   - 方式 B：在测试机 `/etc/hosts` 中写 `<diting-host-ip> api.mycompany.com`。
3. **Agent 配置**：Agent 使用上述 DNS（或 hosts），不配置 HTTP 代理；请求 `https://api.mycompany.com/...` 时解析到 Diting 所在主机，由 Diting 或前置转发并处理。
4. **验证**：同 Proxy，通过审计或 trace_id 确认请求经 Diting。

## 注意事项

- DNS 模式要求 Diting（或前置）对外提供 80/443，且 TLS 若需证书需自行配置。
- 架构上「流量到达 diting-proxy 后的处理链路」与接入方式无关；双接入仅影响流量如何到达 proxy。
- 当前仓库 `deployments/coredns/Corefile` 为示例，可按需修改 `hosts` 中的域名与 IP 以适配 Diting 所在环境。
