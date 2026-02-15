# 谛听 Issue List

**维护说明**：本列表基于 BMAD 实现与架构偏差分析（`_bmad-output/planning-artifacts/bmad-implementation-and-architecture-drift.md`）、`next-steps-roadmap.md` 及紫微技术方案整理。优先级 P0=阻塞/一致性问题，P1=产品化体验，P2=增强与扩展，P3=可选/长期。

---

## 一、命名与 BMAD 一致性

| ID | 标题 | 优先级 | 说明 |
|----|------|--------|------|
| I-001 | BMAD 配置与文档中 project_name 统一为 diting | P0 | Done（2026-02-13）：config.yaml、PRD、implementation-readiness、architecture 等已统一为 diting。 |
| I-002 | Architecture 目录树示例中仓库名统一为 diting | P0 | Done（2026-02-13）：根目录 diting/；主入口路径已改为 cmd/diting_allinone，无 sentinel-ai 残留。 |

---

## 二、架构文档与现状对齐

| ID | 标题 | 优先级 | 说明 |
|----|------|--------|------|
| I-003 | 在 Architecture 中显式区分「当前形态」与「Growth 目标」 | P0 | Done（2026-02-13）：已有「当前实现形态（2026-02）」；默认本地命令已统一为 go run ./cmd/diting_allinone / make run。 |
| I-004 | Current Status 与 Next Steps 引用 diting 与主入口路径 | P1 | Done（2026-02-13）：current-status-and-next、next-steps-roadmap 已为 diting / cmd/diting_allinone；drift 文档已标完成。 |

---

## 三、产品化与体验（来自 next-steps-roadmap P1）

| ID | 标题 | 优先级 | 说明 | 状态 |
|----|------|--------|------|------|
| I-005 | 审批回复体验：短词「批准/拒绝」+ 自动匹配最近待审批 | P1 | 用户仅回复「批准」或「拒绝」时，自动匹配最近一条待审批请求，无需带 request ID；当前 main.go 轮询逻辑要求消息含 req.ID[:8]。见 next-steps-roadmap P1。 | Done（2026-02） |
| I-006 | 审批超时前可配置提醒（如飞书二次提醒） | P1 | 超时前 N 分钟可向确认人再次发送提醒消息，降低漏批。 | Done（2026-02） |
| I-007 | 飞书消息发送失败时的重试与退避 | P1 | 投递失败时按策略重试（次数与退避可配置），并记审计。 | Done（2026-02） |

---

## 四、多审批人与策略（P2）

| ID | 标题 | 优先级 | 说明 |
|----|------|--------|------|
| I-008 | 多审批人配置与策略（多 approval_user_id，任一/全部通过） | P2 | config 支持多个审批人，投递时群发或按策略选人；规则可为「任一通过即放行」或「全部通过」。 |
| I-009 | 按风险等级或 path 配置不同超时与审批人 | P2 | 不同风险或 path 可配置不同超时时间、不同审批人列表。 |

---

## 五、可观测与运维（P3）

| ID | 标题 | 优先级 | 说明 |
|----|------|--------|------|
| I-010 | 审批历史/统计（通过率、响应时间、按方法/路径分布） | P3 | 从审计日志聚合统计，可输出为报表或简单 Web 页。 |
| I-011 | /health、/ready 增强（飞书 token 可用性、最近审批延迟等） | P3 | 就绪探针可包含依赖健康（如飞书 token、最近一次投递延迟）。 |
| I-012 | 配置热更新（部分配置 SIGHUP 或 API 重载） | P3 | 超时、审批人等部分配置支持热更新，避免重启。 |

---

## 六、目录与构建演进（可选）

| ID | 标题 | 优先级 | 说明 |
|----|------|--------|------|
| I-013 | go.mod 上移至仓库根，核心代码上移 | P3 | 按 Go 规范与前期结论：仓库根 = module 根；cmd/diting 内容上移，保留 cmd/diting_allinone 与 cmd/3af_exec；需同步 Makefile、Dockerfile、air、import 路径。 |
| I-014 | 提供 6 个独立可执行入口（cmd/diting-proxy 等） | P3 | Growth：与 Architecture 一致，为 6 组件各提供 main，便于 docker-compose 多容器与独立扩缩容。 |
| I-015 | docker-compose 多容器编排与端口约定 | P3 | 使用 6 个二进制 + compose 编排，文档约定各服务端口与 env。 |

---

## 七、紫微私有链与 DID（扩展）

| ID | 标题 | 优先级 | 说明 | 状态 |
|----|------|--------|------|------|
| I-016 | 私有链子模块规划与接口设计 | P2 | 紫微技术方案将私有链归属谛听；在 diting 内新增 pkg/chain 或 internal/chain，定义 DID 注册/查询与审计存证/验真 API，与 ziwei 技术方案 3.6 一致。设计见 `_bmad-output/planning-artifacts/chain-submodule-design-I016.md`。 | **Done**（2026-02-13 设计完成并实现） |
| I-017 | 实现最小私有链 + DID/存证 API（一期） | P2 | 链抽象层 + 最小可用链（如日志+Merkle），暴露 POST/GET /chain/did/*、POST/GET /chain/audit/*；依赖 I-016。拆为 **Epic 10**，Stories 10.1～10.6 见 `_bmad-output/planning-artifacts/epics.md` 与 `_bmad-output/implementation-artifacts/10-*.md`。 | **Done**（2026-02-13 Epic 10 完成；config.chain-run.yaml、/chain/health 可用） |
| I-018 | 天枢对接：注册/心跳调用谛听 DID 接口 | P2 | 天枢（tianshu）在智能体注册与心跳流程中调用谛听链上 DID 接口；依赖 I-017。 | **Done**（2026-02-13：E-P7 联调已通过；联调约定见 ziwei/docs/open/technical/E-P7-DID联调验证.md） |

---

## 八、测试与质量

| ID | 标题 | 优先级 | 说明 | 状态 |
|----|------|--------|------|------|
| I-019 | 关键包单测覆盖与 CI 回归 | P2 | policy、cheq、audit、proxy pipeline 等核心包保持或补充单测；`go test ./internal/...` 纳入 CI。 | **Done**（2026-02-15：已补 policy/audit/cheq 单测，Makefile `make test`，.github/workflows/test.yml） |
| I-020 | E2E 与 15 分钟演示脚本可重复执行 | P2 | 端到端脚本与文档在干净环境下可重复跑通，耗时与结果可验证。 | **Done**（2026-02-15：verify_exec.sh 可重复、test.sh 端口 8080+PROXY_PORT、docs/E2E与演示可重复执行说明.md） |

---

## 九、文档与交付物

| ID | 标题 | 优先级 | 说明 | 状态 |
|----|------|--------|------|------|
| I-021 | DEVELOPMENT.md 中仓库名与入口描述统一为 diting | P1 | Done（2026-02-13）：标题改为 Diting；仓库结构含 cmd/diting_allinone 推荐入口；主入口表述一致。 |
| I-022 | 接入检查清单与「流量经 Diting」验证步骤可发现 | P1 | Done（2026-02-13）：README 新增「接入与验收」小节，链接 QUICKSTART 接入清单与 ACCEPTANCE_CHECKLIST。 |

---

## 状态约定

- **Open**：未开始  
- **In Progress**：进行中  
- **Done**：已完成  

I-005、I-006、I-007 已标为 Done（2026-02）；其余未标注的为 Open。实施时可将对应 ID 更新为 In Progress/Done 并注明完成时间或 PR。PM 视角的「后续要做的事」见 `_bmad-output/planning-artifacts/pm-what-to-do-next.md`。

---

*生成自 BMAD 偏差分析与 next-steps-roadmap；紫微技术方案见 `ziwei/docs/open/technical/紫微智能体治理基础设施-技术方案.md`。*
