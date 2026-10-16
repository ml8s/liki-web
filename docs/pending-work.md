# 待办：支付上线收尾与后台总览（交接）

更新时间：2026-10-16 · 基线提交：`85b0739`

本文件只记录**尚未完成**的工作，按优先级排列。完成一项就更新或删除对应条目，不要让它
变成过期清单。

## 已完成（不要重复做）

- `85b0739` 已提交并推送：Dodo 一次性付费 `cancel_url` 绝对化 + 调用前校验相对 URL、
  支付失败先 refresh 再写错误（原来错误被 refresh 清掉，买家只看到按钮闪一下）、
  订阅商品未配置时不下发订阅 offer、页脚两列、标题统一 `<页面名> · Liki`、后台点菜单不闪。
- US 生产已配置 Dodo 订阅商品并重建 web 容器生效（只改 env，未换镜像摘要），
  USD 订阅入口恢复。
- 验证记录：本地端到端 `US 头 = 一次性 2990 + 订阅 1990`、`CN 头 = 仅一次性 7900`，
  清空商品 ID 后 US 的订阅 offer 消失；Dodo live 侧按 App 实际参数创建订阅结账会话成功。
- 根因（供接手人理解，不要再排查一遍）：订阅入口消失不是代码问题——它不是"丢了"，
  而是 `liki-deploy` 的 `DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION` 为空；Dodo 侧订阅商品
  `Liki Chat Sub` 一直存在。真正坏掉的是 USD 一次性付费：线上 `93bcfed` 传的是相对
  `cancel_url`（`/app/orders?...`），Dodo 一律回 `400 cancel_url: must be a valid URL`。

## 1. P0 发布并上线本仓支付修复（影响收款）

现状：US 生产仍运行 `93bcfed`（`2026.10.15.1`），USD 一次性付费仍会 400 失败。
完整门禁流程以 `liki-deploy/docs/RELEASE_RUNBOOK.md` 为准，这里只列顺序：

1. bump `cmd/liki-web/VERSION` → `2026.10.16.1`，提交推送。
2. `RELEASE_VERSION=2026.10.16.1 make candidate`（要求干净树；产出
   `build/candidate/candidate-digests.json`）。
3. liki-deploy 跑 assembly：
   `CANDIDATE_MANIFESTS=<abs>/liki-web/build/candidate/candidate-digests.json make candidate-test`。
4. `make candidate-publish`；`git tag -a v2026.10.16.1 <source-commit>` 并推送；
   dispatch `promote-release.yml`（version / source commit / digests 三项一致）。
5. liki-deploy：`CANDIDATE_MANIFESTS=<同上> DEPLOY_LOCK=env/us.lock make update-release-lock`，
   `make check`，提交推送，等 Config CI 绿。
6. 部署：`GHCR_TOKEN="$(gh auth token)" DEPLOY_ENV=env/us.env LOCK_FILE=env/us.lock make deploy-us`。
7. 终验：`VERIFY_URL=https://liki.hk SMOKE_PROFILE=standard SMOKE_INSECURE=0 make deploy-smoke`。

人工验收：

- USD 买家点一次性付费能拉起 Dodo 结账页，不再 400；订阅同样能拉起。
- 结账失败时卡片显示错误文案，而不是无声无响应。
- 卡片默认选中订阅，且可以切到一次性付费。
- 页脚两列无分组标题、标签标题为 `<页面名> · Liki`、后台点菜单不重渲染闪烁（随同版本上线）。

## 2. P1 让 CN 源站的 Dodo 配置与 US 对齐

现状：订阅入口缺少的根因就是该 env 为空。本机 `liki-deploy/env/cn.env` 已填
`DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION`，但该文件不进仓，CN 源站需要**有 SSH 权限的一方**
部署。

- 步骤：从有权限的机器按 liki-deploy 的 cn 流程部署 web 组件（`make deploy-web-cn` 或等价）。
- 验收：CN 源站容器 env 里订阅商品 ID 非空；CNY 买家仍然只看到一次性（国内走汇付、无订阅）；
  国外买家无论落到哪个源站都能看到订阅。

## 3. P1 同步 staging 的 env secret

现状：`Deploy staging` workflow 的 env 来自 GitHub environment `staging-us` 的 secret
`STAGING_US_ENV_FILE_B64`，**不是**仓库里的 `env/staging.env`。本机 `env/staging.env` 已更新。

- 步骤：用更新后的 `env/staging.env` 重新 base64，更新该 secret；以 `apply=false` 跑一次
  `Deploy staging` 复查渲染结果。
- 验收：workflow 渲染出的 `DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION` 非空。

## 4. P2 后台总览页没有数据（**未决，先讨论再动手**）

现状：`web/app/src/views/admin/AdminOverviewView.vue` 只渲染能力标签；后端没有总览接口，
所以总览页没有任何数据。方案未定，**不要先写接口**。

需要先定的事：

- 指标范围：今日 / 近 7 日 / 累计。
- 是否按币种拆分（CNY 汇付 / USD Dodo），退款是否计入 GMV。
- 订阅口径：活跃订阅数、新增、续费、取消。
- 权限：不同管理能力的人看到的总览是否不同。

方向（未定，仅供讨论参考）：新增单个 `GET /api/admin/overview` 按能力返回聚合，前端一次
请求渲染，而不是前端拼多个分页接口。

## 附：口径与真值（避免再次踩坑）

- 路由规则：国外 = Dodo（一次性 + 订阅），国内 = 汇付（只有一次性）。CNY 没有订阅价格，
  所以订阅永不展示给 CNY 买家。
- 价格真值与 Dodo 商品必须一致，改价两边同时改：
  - `Liki Chat`（一次性）2990 USD 分 ↔ `LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS=2990`。
  - `Liki Chat Sub`（订阅）1990 USD 分/月、含税 ↔ `LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS=1990`
    （该变量在各环境由 compose 默认值提供，env 文件不重复声明）。
- 本轮在 Dodo live 建过一个未支付的结账会话（验证用，未扣款，会自然过期）。
