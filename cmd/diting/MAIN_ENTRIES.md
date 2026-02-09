# 入口说明（已清理）

---

## 推荐入口（唯一）

| 入口 | 说明 |
|------|------|
| **cmd/diting_allinone/main.go** | **唯一推荐**。`make build` 编译为 `bin/diting`，使用 config.yaml + .env，集成策略、CHEQ、飞书投递、审计。 |

---

## 备用入口（可选）

同目录下仍保留 **main.go**，使用 config.yaml + .env，需**单独指定文件**编译（与 allinone 不同包）：

| 文件 | 用途 |
| --- | --- |
| **main.go** | 早期代理 + 飞书轮询 + LLM。编译：`go build -o bin/diting_main main.go`（需同目录其他依赖 .go） |

**main_feishu.go + feishu_listener.go** 已迁到 **\_temp/diting/** 作临时/备用，不提交；若需飞书消息回复模式可在该目录单独编译运行。

---

## 已删除/迁出的入口（2025 清理）

以下已移除或迁至 _temp，不再作为正式入口：main_feishu_chat、main_ws、main_ws_fixed、main_complete、main_ollama、main_sdk、main_official_sdk、main_correct_sdk、main_final、main_simple_ws；main_feishu、feishu_listener 已迁 _temp/diting/。
