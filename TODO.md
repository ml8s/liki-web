# TODO

未完成的收尾工作，按优先级排列。做完一项就删掉对应条目。
基线：`85b0739`（2026-10-16）。发布门禁流程见 `liki-deploy/docs/RELEASE_RUNBOOK.md`。

## 已完成，别重复做

- [x] `85b0739`：Dodo 一次性付费 `cancel_url` 绝对化并在调用前校验；结账失败先 refresh 再写错误
      （原来错误被 refresh 清掉，买家只看到按钮闪一下）；订阅商品未配置时不下发订阅 offer；
      页脚两列、标题 `<页面名> · Liki`、后台点菜单不重渲染闪烁。
- [x] US 生产配置 Dodo 订阅商品并重建 web 容器（只改 env，镜像摘要未变），USD 订阅入口恢复。

## P0 发布支付修复

线上 US 仍运行 `93bcfed`（2026.10.15.1），给 Dodo 传的是相对 `cancel_url`，USD 一次性付费一律
`400 cancel_url: must be a valid URL`，影响收款。

- [ ] bump `cmd/liki-web/VERSION` → `2026.10.16.1`，提交推送
- [ ] `git tag -a v2026.10.16.1 && git push origin v2026.10.16.1`（release.yml 自动 gate→构建→签名→发布）
- [ ] liki-deploy dispatch `deploy.yml`（region=us，不带 lock_ref）：自动拉 release 资产组 us.lock →
      assembly e2e → 部署 → 公网 smoke → lock 回写 main
- [ ] 终验：`VERIFY_URL=https://liki.hk SMOKE_PROFILE=standard SMOKE_INSECURE=0 make deploy-smoke`
- [ ] 人工验收：USD 一次性与订阅都能拉起 Dodo 结账页；失败有错误文案而不是无声无响应；
      卡片默认订阅且可切到一次性；页脚与标签标题符合预期

## P1 CN 源站的 Dodo 配置对齐

订阅入口消失的根因是 `liki-deploy/env/cn.env` 的 `DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION` 为空。
本机已填，但该文件不进仓，CN 源站需要有 SSH 权限的一方部署。

- [ ] 从有权限的机器部署 web 组件（`make deploy-web-cn` 或等价流程）
- [ ] 验收：CN 源站容器 env 里订阅商品 ID 非空；CNY 买家仍只看到一次性；USD 买家落在任一源站
      都能看到订阅

## P1 staging 的 env secret

`Deploy staging` 的 env 来自 GitHub environment `staging-us` 的 secret
`STAGING_US_ENV_FILE_B64`，不是仓库里的 `env/staging.env`（本机已更新）。

- [ ] 用更新后的 `env/staging.env` 重新 base64 并更新该 secret
- [ ] 以 `apply=false` 跑一次 `Deploy staging`，确认渲染出的订阅商品 ID 非空

## P2 后台总览页没有数据（未决，先讨论再动手）

`web/app/src/views/admin/AdminOverviewView.vue` 只渲染能力标签，后端没有总览接口，所以总览页
没有任何数据。方案未定，不要先写接口。

- [ ] 先定口径：指标范围（今日 / 近 7 日 / 累计）；是否按币种拆分、退款是否计入 GMV；
      订阅口径（活跃 / 新增 / 续费 / 取消）；不同管理能力是否看到不同视图
- [ ] 再定方案（参考方向：单个 `GET /api/admin/overview` 返回聚合，前端一次请求渲染）

## 口径备忘

- 路由规则：国外 Dodo（一次性 + 订阅），国内汇付（只有一次性）；CNY 没有订阅价格，
  所以订阅永不展示给 CNY 买家。
- 价格真值必须与 Dodo 商品一致，改价同时改：`Liki Chat`（一次性）2990 USD 分 ↔
  `LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS=2990`；`Liki Chat Sub`（订阅）1990 USD 分/月含税 ↔
  `LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS=1990`（各环境由 compose 默认值提供，env 不重复声明）。
- 本轮在 Dodo live 建过一个未支付的结账会话（验证用，未扣款，会自然过期）。
