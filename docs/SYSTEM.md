# Liki 系统全况与本仓对齐

> 本文档描述 Liki 多 Agent 系统的全局架构，以及本仓（产品 BFF 仓）在其中
> 的角色与对齐点。让本仓负责人知道：我产什么、谁消费、边界在哪、变更如何传播。

## 一、系统全局架构

Liki 是多 Agent 命理专家系统，四仓分工：

```text
浏览器 → Caddy → liki-web (BFF)
                   └─AG-UI→ liki-agents (装配镜像: 运行时+组合工件) ──MCP──> engine-mcp / counsel-mcp
                              router → 全功能领域专家委派，审计留痕
```

| 仓 | 角色 | 产物 |
| --- | --- | --- |
| liki | 内容/领域仓 | 专家内容、engine/counsel MCP、deployment 工件、装配镜像、web skill bundle |
| liki-agents | 通用运行时仓 | 纯运行时镜像、AgentDeployment 运行时校验 |
| **liki-web（本仓）** | 产品 BFF 仓 | **liki-web 镜像（BFF + 前端）** |
| liki-deploy | 发布编排仓 | 纯清单：compose + env + Caddyfile，只 pin 版本 |

## 二、本仓角色（liki-web BFF 仓）

本仓是**产品前端 + BFF**，用户面对系统的全部产品能力：

- 用户/认证/权益/订单/支付/档案/会话持久化
- **Liki Chat 产品**：通过 AG-UI 网关（`/api/chat/sessions/{id}/ag-ui`）调用 liki-agents 运行时
- 托管客户端 skill（`web/skills/liki`）供终端用户下载

**Agent 模式（唯一）**：所有 chat 会话（含 profile-bound）都走 liki-agents，local 进程内 runner 已退役。

## 三、关键对齐点（负责人必读）

### 3.1 消费 liki 的 skill bundle（不 checkout 源码）

`.github/actions/prepare-skills` 从 **exact ml8s/liki release asset** 下载
`liki-web-skill-bundle.tar.gz`，先校验 SHA256，再解包并核对内嵌 archive digest：

```text
liki release bundle → 校验 SHA256 / archive digest
                    → 解包 liki.tar.gz 为 web/skills/liki/
                    → 叠加 webapp + web/skills/liki.tar.gz + discovery index
```

- **不再 checkout liki 源码、不再本地 build-archive**（消除跨仓源码依赖）
- 需要更新 skill 内容时，**先发 liki release**，再构建本仓镜像
- CI 必须提供 `LIKI_SKILLS_RELEASE_TAG` 与 `LIKI_SKILLS_BUNDLE_SHA256`

### 3.2 Agent 运行时

- local 进程内 runner 已删除，Liki Chat 唯一路径是 liki-agents AG-UI
- 必需 env：`LIKI_AGENTS_URL`、`LIKI_AGENTS_INTERNAL_TOKEN`；模型 provider 凭据只属于 liki-agents，不得进入 liki-web runtime
- `cmd/liki-web/main.go`：`SetAgentRunner` 把 AG-UI 客户端挂到 chat service；readiness 校验 agents 端点

### 3.3 镜像发布（release 触发）

`docker-publish` job：`make build-sitemap build-go build-web` → 打 `liki-web:<version>` 镜像 push GHCR。

- 镜像依赖预构建产物（bin + web/dist + web/app/dist + web/skills）
- **web/skills 来自 prepare-skills**（消费 liki bundle），非本仓源码

### 3.4 与 liki-agents 的协议边界

- 本仓 → liki-agents：**AG-UI**（`internal/liki/agui_runner.go`）
- 本仓只通过 AG-UI 与运行时通信，**不感知** agent 内部拓扑（router/专家）
- 会话绑定/权益/档案写回在本仓做（不泄漏给运行时）

## 四、变更如何传播（负责人须知）

| 本仓变更 | 影响 | 传播路径 |
| --- | --- | --- |
| 改前端/BFF | 产品功能 | 发本仓镜像 → deploy 换 tag |
| 改 chat 产品逻辑 | chat 体验 | 发本仓镜像（不改 agents/内容） |
| 依赖 skill 内容更新 | 前端 skill 页面/客户端下载 | **先发 liki release** → 重发本仓镜像 |

**对齐提醒**：skill 内容（SKILL.md/app 卡/webapp）的源在 liki 仓；本仓只镜像打包。若本仓需要改 skill 页面，改动须回 liki 仓再走 release。

## 五、本仓 CI

- `check`/`test`/`e2e`：质量门（全部依赖 `prepare-skills` 消费 bundle）
- `docker-publish`：release 触发 push `liki-web`

## 六、相关文档

- `ARCHITECTURE.md`：本仓内部架构
- `docs/mcp.md`、`docs/database/`、`docs/aipay.md`：BFF 实现细节
- `liki/docs/SYSTEM.md`：内容仓视角的全况
- `liki-agents/docs/SYSTEM.md`：运行时仓视角的全况
- `liki-deploy/docs/ARCHITECTURE.md`：系统架构与发布时序
