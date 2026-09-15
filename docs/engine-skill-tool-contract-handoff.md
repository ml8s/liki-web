# 任务：对齐 Liki 工具结果契约（引擎 / skill 侧）

> **状态：已审计并结案（无需改动）。** 审计 `../liki/skills` 全部提示词（SKILL.md + app/*.md）后确认：**零信封引用**（无 `_product` / `ok` / `data` / `content` / 返回形状描述）；python 层读引擎 `["data"]` 因引擎 RPC 契约保持不变而无需改。目标「LLM 拿到命理本体」已由 liki-web 在 agent 边界解包达成，引擎/skill 侧不需要任何代码改动。本文保留作为背景与决策记录。

> 这份是交接给引擎/skill 维护者的提示词。liki-web 仓库侧另按
> `docs/backend-tool-consistency-plan.md` 执行，两边以本文「目标契约」对齐。

## 背景

Liki 把 agent 视为平台 API 的又一个客户端。为让 LLM 拿到的是「命理结果本体」而非通用 RPC 传输壳，正在统一 agent 工具结果契约。你负责 **`../liki`（引擎 + skills）** 侧对齐；liki-web 侧由主工程师负责，你**不要动 liki-web**。

## 目标契约（不要向后兼容）

- **成功**：所有工具结果 = 命理结果本体（本命盘 `pan` / 断语 / 定盘结果 / 文件内容），**不带** `_product`、`ok`、`data`、`content` 这类传输包裹。
- **失败**：错误统一由调用方（liki-web `agent.Execute`）包成 `{"error": "..."}`；工具自身不包成功/失败信封。
- 领域识别靠**工具名**（`bazi.*` / `full_paipan` / `qimen.*` …），不依赖信封标签。

## liki-web 侧（主工程师负责，你只对齐、不修改）

- liki-web 的引擎工具 handler 在 agent 边界把引擎结果 `{"_product":..., "data":...}` 解包为裸 `data` 喂给 LLM。
- `read_file` 改为返回裸文件内容。
- python executor 已消费 `agent_cli.py` 的 `{"ok":true,"data":...}` 并解包。
- **结论**：LLM 看到的已是裸命理本体；引擎的传输形状只影响内部消费者，不影响 LLM。

## 你需要在 ../liki 做的事

### A. Skill 提示词审计（必须）
审计会被 LLM 读取的提示词（不只 SKILL.md，还有 `app/*.md` 等）：
- 目标：删除/改写任何假设工具结果带 `_product` / `data` 字段 / `ok` / `content` 信封的描述；改为「工具直接返回命理结果（盘 / 断语）」。
- 先 grep 确认：`_product`、`"data"`、`"ok"`、`"content"`。若提示词从未引用信封形状，则无需改。
- 覆盖：`skills/liki/` 统一 skill 内的 `bazi/`、`divination/` 域工具目录及命名/风水提示词。

### B. python 工具传输约定（核对，通常无需改）
- `skills/*/tools/agent_cli.py` 的 `{"ok":true,"data":...}` / `{"ok":false,"error":...}` 是 liki-web executor 与 CLI 的**进程内传输协议**，liki-web 已消费——**不要改**。
- `paipan.py` 等调用引擎时读 `result["data"]`——**保持**（引擎 RPC 契约不变，见 C）。

### C. 引擎 RPC 契约（推荐：保持不动）
- `liki/engine` 的 `wrapResult` 产出 `{"_product":..., "data":...}` 是**公共 RPC 契约**，被命名 / 报告 / 合盘等多个消费方与引擎测试依赖。
- **推荐**：保持不变；由 liki-web 在 agent 边界解包，已达成 LLM 侧「命理本体」目标。
- **若你方坚持彻底命理向**（引擎直接返回裸结果），需一并改：
  1. `engine/internal/agent/types.go` 的 `wrapResult`；
  2. 所有引用 `["data"]` 的引擎测试（`rpc_registry_test.go`、`rpc_robustness_test.go`、`schema_full_test.go` 等）；
  3. `skills/*/tools/agent_cli.py`、`paipan.py` 的取数逻辑。
  - ⚠️ 此选项爆炸半径大、跨消费方，需你方评估后决定，并在引擎 CHANGELOG 标注。**我的建议是不做。**

## 验收

- `../liki/skills` 提示词层面 grep `_product` 无命中。
- SKILL.md 对工具结果的描述与新契约一致（工具直接返回命理本体）。
- 若选了 C 的彻底版：引擎全量测试绿 + python 取数逻辑同步 + 跨消费方回归。
- 不动 liki-web；不引入新的向后兼容分支。

## 关联

- liki-web 侧方案：`liki-web/docs/backend-tool-consistency-plan.md`