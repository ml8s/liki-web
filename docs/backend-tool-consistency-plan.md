# Agent 工具切面一致性优化方案

- 版本：v1.1（已实施）
- 关联：`docs/platform-fs.md`、`docs/backend-profiles-plan.md`
- 原则：**不向后兼容**，向命理域方向统一（工具交付"命理本体"，而非通用 RPC 传输壳）

## 实施结果

| 项 | 状态 |
|---|---|
| I1 命理向统一结果契约 | ✅ liki-web 边界解包：引擎 `{_product,data}`→裸 `data`；read_file 返回裸内容；python executor 本已裸；错误 Execute 统一 `{"error":...}` |
| I2 统一工具超时 | ✅ `agent.Execute` 每次工具调用外层 `toolCallTimeout=30s` |
| I3 429 语义化 | ✅ `update_profile_birth_time` 429 → "操作太频繁，请稍后重试"；409 保留；其余附 error code |
| I4 会话绑定泛化 | ✅ 公共 guard `requireBoundProfile(ctx)`，写工具统一前置 |
| I5 命名 | ✅ 已一致（snake_case），不改 |
| I6 注册来源文档化 | ✅ 见下方"工具注册指南" |
| I7 read_file 例外 | ✅ 见下方 |

**引擎 / skill 侧结论**：审计 `../liki/skills` 全部提示词（SKILL.md + app/*.md）——**零信封引用**；python 层读引擎 `["data"]` 因引擎 RPC 契约不变而无需改。**引擎/skill 无需任何代码改动**，目标已由 liki-web 边界解包达成（保持公共 RPC 契约稳定，跨消费方零风险）。

## 工具注册指南（I6）

| 工具类型 | 来源 | 注册位置 | 如何新增 |
|---|---|---|---|
| 引擎 RPC | `rpc.discover` 懒发现 | `internal/agent/tools.go` `discoverEngineTools` | 引擎加 RPC 即自动出现；scope 用 `Prefixes`（如 `bazi.`）放行 |
| python 规则工具 | `skills/<skill>/tools/skill-tools.json` | `internal/agent/skill_tools.go` `RegisterSkills` | 加 skill-tools.json 条目；scope 用 `Exact` 放行 |
| 服务端工具 | 静态 Go | `cmd/liki/main.go` 各 runner 注册 | `tools.Register(...)`；写工具须走 `requireBoundProfile` guard，并加入 `SkillToolScopes[skill].Exact` |

可见性统一由 `agent.SkillToolScopes`（deny-by-default）兜底，三类来源一致。

## read_file 例外（I7）

`read_file` 是**唯一非网络工具**：本地读 `web/skills/<skill>/` 下的提示词/领域文件，只读、带路径逃逸防护（`filepath.IsLocal` + 前缀校验）。受控例外，不违反"工具走权威服务"原则（它读的是 skill 自带静态文件，非用户/平台数据）。

## 0. 现状审计（切面不一致清单）

| 切面 | 现状 | 不一致点 |
|---|---|---|
| 成功信封 | 引擎 `{_product,data}` / python 裸 data / read_file `{content}` / update `{status}` | **I1** 四套形状 |
| 失败信封 | Execute 统一 `{"error":...}` | 已一致 ✅ |
| 超时 | python 30s / 引擎依赖外层 ctx(120s) / update 10s | **I2** 不统一 |
| 429/限流 | update 撞 429 返回裸状态码 | **I3** 无语义化 |
| 身份 | 会话绑定只接 update 一个工具 | **I4** 单点接线 |
| 命名 | 自定义工具 snake_case（与 python 一致） | 已一致 ✅ |
| 注册来源 | rpc.discover / skill-tools.json / 静态 Go | **I6** 需文档化 |
| 网络性 | read_file 是唯一非网络工具 | **I7** 需明示受控例外 |

## 1. I1 · 命理向统一结果契约（不向后兼容）

**目标**：工具结果 = 命理本体，无传输壳。

**统一契约**：
```
成功：工具直接返回命理结果本体
      （本命盘 pan / 断语 / 定盘结果 / 文件内容 / 档案更新结果）
失败：Execute 层统一 → {"error": "<message>"}
```

**各工具归一**：
| 工具 | 现状 | 改为 |
|---|---|---|
| 引擎 RPC（`callEngine`） | 返回 `{_product, data}` | 解包 → 只留 `data`（盘本体） |
| python 工具（executor） | 已返回裸 `data` | 不变 ✅ |
| `read_file` | `{"content":"..."}` | 返回裸文件内容字符串 |
| `update_profile_birth_time` | `{status,...}` | 不变 ✅（已是裸） |

**领域识别不依赖信封**：工具名自带领域（`bazi.*`→八字、`full_paipan`→八字、`qimen.*`→奇门、`read_file`→文本、`update_profile_birth_time`→档案），LLM 知道自己在调哪个工具，无需 `_product` 壳。

**爆炸半径**：
- 仅 liki-web 内部：`agent.Execute` 的引擎工具 handler 解包、read_file 返回形状、agent 测试
- **skill prompt 不引用信封**（已 grep 确认），外部 liki 仓无需改

**附：可选项（评审时定）**——是否保留一个轻量 `kind` 标签：
```
{ "kind": "bazi", "data": { ... } }
```
默认**不加**（工具名已带领域，保持最裸）。若你希望显式带领域标签，评审时选择"加 kind"。

## 2. I2 · 统一工具调用超时
- `agent.Execute` 每次 `a.tools.Execute` 外层统一 `ToolCallTimeout = 30s`（与 python 一致）。
- 内层超时（python 30s / update 10s）保留，更小者生效。
- 常量定义于 `internal/agent`。

## 3. I3 · 429/限流语义化
- `update_profile_birth_time` 解析平台 API 错误信封：
  - HTTP 429 或 `code=rate_limited` → `"操作太频繁，请稍后重试"`
  - HTTP 409 `profile_conflict` → 现有"档案刚被其他设备修改"（保留）
  - 其余非 2xx → 附状态码

## 4. I4 · 会话绑定泛化
- 提取公共 guard：`requireBoundProfile(ctx) (binding, token string, err error)`
  - 校验绑定存在 + 会话 token 存在 + 返回 identity
- 所有**服务端写工具**必须走此 guard；文档标注"写工具统一前置"。
- 本轮只有一个写工具，guard 先立规矩，未来加写工具不重复。

## 5. I5 · 命名（确认一致，不改）
- 自定义工具 `update_profile_birth_time` 已用 snake_case，与 python 工具（`full_paipan` 等）一致。引擎 dot 命名属引擎契约，不改。

## 6. I6 · 工具注册来源文档化
- `docs/backend-profiles-plan.md` 增"工具注册指南"：三类工具（引擎 discover / python skill-tools.json / 静态 Go）各自注册位置、如何新增、scope 白名单配置。

## 7. I7 · read_file 例外明示
- 文档标注：`read_file` 是唯一非网络工具（本地读 skill 目录，只读、路径逃逸防护），为受控例外。

## 8. 评审结论（自审）
- 方向：I1 向命理域统一（去 `_product`/`ok`/`content` 传输壳），符合"agent=解读黑盒、工具交付命理本体"。
- 风险：引擎 `_product` 粒度丢失 → 由 `data` 内部字段 + 工具名兜底；agent 测试需同步。
- 范围：全在 liki-web 内部，不碰引擎契约、不碰 skill 仓、不碰前端。

## 9. 实施顺序与测试
| 步 | 内容 | 验证 |
|---|---|---|
| 1 | I2 统一超时 + 常量 | agent 测试 |
| 2 | I3 429 语义化 + I4 guard 泛化 | chat 工具测试 |
| 3 | I1 信封归一（引擎解包/read_file 裸返回/Execute 错误统一） | agent 测试更新 + 全量 go test |
| 4 | I6/I7 文档 | 无 |

**最终验证**：`go build` / `go vet` / `go test ./...` + 前端不受影响（typecheck/test/build 抽查）。