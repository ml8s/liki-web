# Liki 推广伙伴带货系统 — 设计文档

> 状态：后端最小闭环已实现；Partner Portal / 提现 / 退款待实施
> 更新：2026-09-20

---

## 一、业务模型

### 产品

Liki 起名（核心产品）

| 层 | 内容 | 价格 |
|---|---|---|
| 免费 | 候选名预览（喜用五行 + 单字/双字候选字） | ¥0 |
| 付费 | Chat 定制化起名（AI 顾问结合八字完整起名） | **¥88** |

### 两层角色

| 角色 | 目标人群 | 做什么 | 收入来源 | 佣金 |
|---|---|---|---|---|
| **招商** | 创业博主 | 招募 KOL（有网络、想建团队） | KOL 产生订单 | 20%（¥17.6/单） |
| **推广** | 命理博主、宝妈博主 | 发内容带货（有粉丝、会创作） | 自己带来订单 | 50%（¥44/单） |

### 准入条件

**推广大使必须是付费用户**：购买 Liki Chat ¥88（使用过产品）才能开通推广资格。确保推广内容基于真实体验。

### 分销模式

```
KOL 注册 → 付费 ¥88（使用产品）→ 开通推广资格
KOL 发小红书/抖音/YouTube 内容，挂专属链接
用户点击 → 进 liki.hk → 填邮箱登录 → 终身挂在该 KOL 名下
用户付费 ¥88 → 系统自动：归因 + 计算佣金 + 看板 + 提现
```

**没有**：裂变、培训、证书、第三层佣金。

### 经济账（¥88 订单）

**KOL 被平台直接招募（无招商）**：
```
端用户付费 ¥88
├── KOL  50%  = ¥44
└── 平台 50%  = ¥44
```

**KOL 被招商招募**：
```
端用户付费 ¥88
├── KOL  50%  = ¥44
├── 招商 20%  = ¥17.6
└── 平台 30%  = ¥26.4
```

**招商自己也带货**：
```
端用户付费 ¥88
├── 招商 50%  = ¥44（直接带货）
└── 平台 50%  = ¥44
```

### 市场对标

| 角色 | 本方案 | 淘宝客/京东联盟 | 评价 |
|---|---|---|---|
| KOL | 50%（¥44） | 10-30% | 偏高，新平台需要吸引力 |
| 招商 | 20%（¥17.6） | 5-15%（团长） | 合理偏高 |
| 平台 | 30%（¥26.4） | 60-85% | 偏薄，数字产品边际成本低 |

### 核心指标

- 点击→注册转化率
- 注册→付费转化率
- KOL 平均佣金/月
- 招商平均招募 KOL 数
- 提现周期

---

## 二、架构

### 三套前端，一个域名

```
liki.hk
├── /app/*        主站 Vue（起名/Chat/排盘/建档）— 端用户
├── /partners/*   独立 Vue（链接/数据/佣金/提现）— KOL & 招商
├── /en/*         营销站 astro（落地页/定价/登录）
├── /             重定向到 /en/
```

### 体验隔离

- **端用户**：在 `/app` 里完成所有操作。永远看不到佣金、推广、分成。
- **KOL & 招商**：通过营销站的「成为推广伙伴」入口进入 `/partners`。只看到自己的推广/招商工具。
- **营销站**：公共面——定价页有「成为推广伙伴」CTA。

### Caddy 路由

```
/app/*        → main Vue build
/partners/*   → partner Vue build
/en/*         → astro build
```

### 后端

新增 `internal/referral/` 包，同时服务两套前端。和现有 `internal/payment/`、`internal/auth/` 同级。

---

## 三、数据模型

### referrers（推广伙伴）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | bigserial | PK | |
| user_id | bigint | UNIQUE, FK users | 关联用户 |
| code | text | UNIQUE | 专属码，如 `MING_abc123` |
| rate | decimal(5,2) | NOT NULL, DEFAULT 50.00 | 佣金率（%） |
| recruiter_id | bigint | FK users, NULLABLE | 招到他的人（招商） |
| role | text | NOT NULL, DEFAULT 'kol' | kol / zhaoshang / both |
| status | text | NOT NULL, DEFAULT 'active' | active / frozen |
| created_at | timestamptz | NOT NULL, DEFAULT now() | |

### referral_attributions（归因绑定）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| user_id | bigint | PK, FK users | 一个用户只归一个伙伴 |
| ref_code | text | NOT NULL | 归属的伙伴码 |
| referred_at | timestamptz | NOT NULL, DEFAULT now() | |

### referral_commissions（佣金记录）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | bigserial | PK | |
| order_id | bigint | FK orders | 关联订单 |
| user_id | bigint | FK users | 付费用户 |
| referrer_user_id | bigint | FK users | 推广伙伴 |
| amount_minor | bigint | NOT NULL | 实付金额（分） |
| currency | text | NOT NULL | CNY / USD |
| rate | decimal(5,2) | NOT NULL | 佣金率（%） |
| commission_minor | bigint | NOT NULL | 佣金（分） |
| status | text | NOT NULL, DEFAULT 'pending' | pending / paid / voided |
| created_at | timestamptz | NOT NULL, DEFAULT now() | |
| paid_at | timestamptz | NULLABLE | |

### referral_payouts（提现记录）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | bigserial | PK | |
| referrer_user_id | bigint | FK users | |
| amount_minor | bigint | NOT NULL | |
| currency | text | NOT NULL | |
| method | text | NOT NULL | alipay / paypal |
| account | text | NOT NULL | 账号 |
| status | text | NOT NULL, DEFAULT 'pending' | pending / processing / completed / failed |
| created_at | timestamptz | NOT NULL, DEFAULT now() | |
| processed_at | timestamptz | NULLABLE | |

### 索引

```sql
CREATE UNIQUE INDEX idx_referrers_user_id ON referrers(user_id);
CREATE UNIQUE INDEX idx_referrers_code ON referrers(code);
CREATE INDEX idx_referrers_recruiter_id ON referrers(recruiter_id);
CREATE INDEX idx_referral_attributions_ref_code ON referral_attributions(ref_code);
CREATE INDEX idx_referral_commissions_user_id ON referral_commissions(user_id);
CREATE INDEX idx_referral_commissions_referrer_user_id ON referral_commissions(referrer_user_id);
CREATE INDEX idx_referral_commissions_order_id ON referral_commissions(order_id);
CREATE INDEX idx_referral_commissions_status ON referral_commissions(status);
CREATE INDEX idx_referral_payouts_referrer_user_id ON referral_payouts(referrer_user_id);
```

---

## 四、API 设计

### 新增端点（`internal/referral/handler.go`）

#### 1. 开通推广伙伴

```
POST /api/referrals/me
Authorization: Bearer <token>

{
  "role": "kol"    // kol / zhaoshang / both
}

Response 201:
{
  "id": 1,
  "code": "MING_abc123",
  "rate": 50.0,
  "role": "kol",
  "status": "active"
}
```

逻辑：
- 校验：用户已付费（有 ProductChat 或 ProductNaming 的有效订单）
- 检查用户是否已有 referrer 记录，有则返回 409
- 生成唯一 code（格式：`MING_` + 8位随机字母数字）
- 写入 `referrers` 表
- 返回创建的伙伴记录

#### 2. 推广数据总览

```
GET /api/referrals/me/summary
Authorization: Bearer <token>

Response 200:
{
  "code": "MING_abc123",
  "rate": 50.0,
  "role": "kol",
  "visits": 120,
  "signups": 35,
  "paid": 8,
  "total_commission": 70400,
  "paid_commission": 44000,
  "pending_commission": 26400,
  "recruited_kols": 5,
  "recruited_kol_commission": 8800
}
```

逻辑：
- `visits`：来自 tracking（v1 显示 0 或占位）
- `signups`：`referral_attributions` 中 `ref_code` 匹配的数量
- `paid`：`referral_commissions` 中 `status != 'voided'` 的 distinct `user_id` 数
- `total_commission`：佣金总额（分）
- `paid_commission`：已提现佣金
- `pending_commission`：待提现佣金
- `recruited_kols`：招商招到的 KOL 数量（仅 role=zhaoshang/both 显示）
- `recruited_kol_commission`：招商从 KOL 获得的佣金总额

#### 3. 佣金明细

```
GET /api/referrals/me/commissions?page=1&size=20
Authorization: Bearer <token>

Response 200:
{
  "items": [
    {
      "id": 1,
      "order_id": 100,
      "amount_minor": 8800,
      "currency": "CNY",
      "rate": 50.0,
      "commission_minor": 4400,
      "status": "pending",
      "source": "direct",
      "created_at": "2026-09-18T10:00:00Z"
    }
  ],
  "total": 8
}
```

#### 4. 申请提现

```
POST /api/referrals/me/payouts
Authorization: Bearer <token>

{
  "amount_minor": 5000,
  "currency": "CNY",
  "method": "alipay",
  "account": "xxx@alipay.com"
}

Response 201:
{
  "id": 1,
  "amount_minor": 5000,
  "currency": "CNY",
  "method": "alipay",
  "status": "pending"
}
```

逻辑：
- 校验：余额（pending_commission）≥ 申请金额
- 校验：金额 ≥ 最低门槛（¥50 / $10）
- 写入 `referral_payouts` 表
- 人工打款后更新 status

#### 5. 招商查看招募的 KOL（仅招商角色）

```
GET /api/referrals/me/kols?page=1&size=20
Authorization: Bearer <token>

Response 200:
{
  "items": [
    {
      "user_id": 200,
      "code": "MING_xyz789",
      "created_at": "2026-09-10T10:00:00Z",
      "total_orders": 12,
      "total_commission": 52800
    }
  ],
  "total": 5
}
```

#### 6. 校验专属码（公开，无需登录）

```
GET /api/referrals/landing?code=MING_abc123

Response 200:
{
  "valid": true,
  "code": "MING_abc123"
}

Response 404:
{
  "valid": false
}
```

### 改动端点

#### POST /api/auth/email/verify（加 ref 参数）

```
POST /api/auth/email/verify

{
  "email": "user@example.com",
  "code": "123456",
  "ref": "MING_abc123"
}
```

逻辑改动（在现有逻辑后追加）：
```
if ref 参数存在:
    查询 referrers 表，确认 code 有效且 status='active'
    如果用户是新用户（刚创建）:
        写入 referral_attributions（user_id, ref_code, referred_at）
```

---

## 五、佣金触发（后端 hook）

### fulfillment.go 改动

在现有 fulfillment 逻辑成功后追加：

```
func OnFulfillmentSuccess(ctx, orderID, userID, amountMinor, currency):
    // 1. 查该用户的归因
    attr = referralStore.GetAttribution(ctx, userID)
    if attr == nil: return

    // 2. 查 KOL 的伙伴记录
    kol = referralStore.GetReferrer(ctx, attr.RefCode)
    if kol == nil || kol.Status != 'active': return

    // 3. 计算 KOL 佣金（50%）
    kolCommission = amountMinor * kol.Rate / 100

    // 4. 写 KOL 佣金记录
    referralStore.CreateCommission(ctx, {
        order_id, user_id, referrer_user_id: kol.UserID,
        amount_minor, currency, rate: kol.Rate,
        commission_minor: kolCommission, status: 'pending',
        source: 'direct'
    })

    // 5. 如果有招商（kol.recruiter_id != null），计算招商佣金（20%）
    if kol.RecruiterID != null:
        zhaoshang = referralStore.GetReferrerByUserID(ctx, kol.RecruiterID)
        if zhaoshang != nil && zhaoshang.Status == 'active':
            zsCommission = amountMinor * 20 / 100
            referralStore.CreateCommission(ctx, {
                order_id, user_id, referrer_user_id: zhaoshang.UserID,
                amount_minor, currency, rate: 20.0,
                commission_minor: zsCommission, status: 'pending',
                source: 'recruited'
            })
```

### 退款处理

```
func OnOrderRefunded(ctx, orderID):
    referralStore.VoidCommissionsByOrder(ctx, orderID)
```

---

## 六、前端 — Partner Portal

### 技术栈

- Vue 3 + TypeScript（和主站同栈）
- 独立构建，部署在 `/partners/*`
- 组件可复用（el-button、el-table 等），但路由和页面独立

### 页面结构

```
/partners/
├── /                    登录页（邮箱+验证码）
├── /register            申请成为推广伙伴（选择角色 + 勾协议 + 付费开通）
├── /dashboard           推广数据总览
├── /referrals           专属链接 + 种草文案模板
├── /commissions         佣金明细列表
├── /payouts             提现申请 + 历史
└── /kols                招商查看招募的 KOL（仅招商角色可见）
```

### 页面设计

#### 登录页 `/partners/`

```
┌──────────────────────────────────┐
│           Liki 推广伙伴            │
│                                  │
│  邮箱：[_______________]         │
│  验证码：[______] [发送验证码]    │
│                                  │
│          [登录]                  │
│                                  │
│  还没有账号？[注册]               │
└──────────────────────────────────┘
```

#### 申请页 `/partners/register`（仅首次登录显示）

```
┌──────────────────────────────────┐
│        成为推广伙伴                │
│                                  │
│  ⚠ 需先购买 Liki Chat（¥88）     │
│    才能开通推广资格                │
│                                  │
│  选择角色：                       │
│  ○ 推广伙伴（发内容带货）         │
│  ○ 招商伙伴（招募推广伙伴）       │
│  ○ 两者都做                      │
│                                  │
│  昵称：[_______________]         │
│                                  │
│  ☑ 我已阅读并同意《推广伙伴协议》  │
│                                  │
│          [一键开通]              │
└──────────────────────────────────┘
```

#### Dashboard `/partners/dashboard`

```
┌──────────────────────────────────┐
│  推广数据总览                      │
│                                  │
│  ┌──────┬──────┬──────┬──────┐  │
│  │ 访问  │ 注册  │ 付费  │ 佣金  │  │
│  │  120  │  35  │   8  │ ¥704 │  │
│  └──────┴──────┴──────┴──────┘  │
│                                  │
│  待提现：¥264.00   已提现：¥440.00│
│                                  │
│  [查看佣金明细]  [申请提现]        │
│                                  │
│  ── 招商数据（仅招商角色）──       │
│  招募 KOL：5 人                   │
│  KOL 佣金：¥88.00                │
└──────────────────────────────────┘
```

#### 专属链接 `/partners/referrals`

```
┌──────────────────────────────────┐
│  你的推广链接                      │
│                                  │
│  liki.hk?ref=MING_abc123        │
│  [复制链接]                       │
│                                  │
│  种草文案模板：                    │
│  ┌────────────────────────────┐  │
│  │ 想给宝宝取个好名字？这个     │  │
│  │ 工具能根据八字推荐候选名     │  │
│  │ 字，还有五行解析 👇         │  │
│  │ liki.hk?ref=MING_abc123   │  │
│  └────────────────────────────┘  │
│  [复制文案]                       │
└──────────────────────────────────┘
```

#### 佣金明细 `/partners/commissions`

```
┌──────────────────────────────────┐
│  佣金明细                         │
│                                  │
│  ┌──────┬──────┬──────┬──────┐  │
│  │ 时间  │ 实付  │ 佣金率 │ 佣金  │  │
│  │ 09-18 │ ¥88  │  50%  │ ¥44  │  │
│  │ 09-17 │ ¥88  │  20%  │ ¥17.6│  │
│  └──────┴──────┴──────┴──────┘  │
│                                  │
│  共 8 笔  合计 ¥352.00           │
└──────────────────────────────────┘
```

#### 提现 `/partners/payouts`

```
┌──────────────────────────────────┐
│  申请提现                         │
│                                  │
│  可提现余额：¥264.00              │
│                                  │
│  提现金额：[_______________]      │
│  提现方式：○ 支付宝  ○ PayPal    │
│  账号：[_______________]          │
│                                  │
│  最低提现 ¥50                     │
│                                  │
│          [申请提现]               │
│                                  │
│  提现历史：                       │
│  ┌──────┬──────┬──────┬──────┐  │
│  │ 时间  │ 金额  │ 方式  │ 状态  │  │
│  │ 09-15 │ ¥100 │ 支付宝 │ 已完成│  │
│  └──────┴──────┴──────┴──────┘  │
└──────────────────────────────────┘
```

#### 招商查看 KOL `/partners/kols`（仅招商角色可见）

```
┌──────────────────────────────────┐
│  我招募的 KOL                     │
│                                  │
│  ┌──────┬──────┬──────┬──────┐  │
│  │ 昵称  │ 加入时间│ 订单数│ 佣金  │  │
│  │ 小红  │ 09-10 │  12  │ ¥528 │  │
│  │ 宝妈A │ 09-12 │   8  │ ¥352 │  │
│  └──────┴──────┴──────┴──────┘  │
│                                  │
│  合计 5 人  佣金 ¥880            │
└──────────────────────────────────┘
```

---

## 七、营销站改动

### `?ref` cookie 写入

在 astro 营销站的所有页面中加入：

```javascript
const params = new URLSearchParams(window.location.search)
const ref = params.get('ref')
if (ref) {
  fetch(`/api/referrals/landing?code=${ref}`)
    .then(r => r.json())
    .then(d => {
      if (d.valid) {
        document.cookie = `liki_ref=${ref}; path=/; max-age=${30*24*3600}`
      }
    })
}
```

### 「成为推广伙伴」入口

在 `/en/pricing` 页底部加 CTA：

```html
<section class="partner-cta">
  <h3>成为推广伙伴</h3>
  <p>购买 Liki Chat 后即可开通，分享专属链接，用户付费后你赚 50% 佣金。</p>
  <a href="/partners">了解详情</a>
</section>
```

---

## 八、合规

### 推广伙伴协议（入驻即签）

核心条款：
1. 内容规范：不承诺结果、不迷信表述、不夸大效果
2. 免责声明：服务仅供文化娱乐参考
3. 违规处理：冻结佣金、取消伙伴资格
4. 黑名单：同邮箱/设备/IP 作弊永久封禁

### 种草文案模板（合规化）

```
想给宝宝取个好名字？这个工具能根据八字推荐候选名字，还有五行解析 👇
[链接]
```

禁止：
- "算得准"、"必灵"、"改运"、"转运"
- 承诺效果
- 迷信表述

### 防刷规则

1. 同邮箱只归一次（已有用户不重新绑定）
2. 伙伴不能给自己归因（IP/设备去重）
3. 同 IP 短时间大量注册触发风控

---

## 九、实施计划

### M1：后端（3-4 天）

- 创建 `internal/referral/` 包（handler/service/store/types）
- 4 张表 migration（含 recruiter_id、role 字段）
- 6 个 API 端点（开通/总览/佣金/提现/KOL列表/校验码）
- `auth/verify` 加 ref 参数
- `fulfillment` 成功后只发布事实；referral intake 持久化 + worker 幂等结算（含招商佣金）
- 补偿 reconcile 扫描 platform 已成功 fulfillment，处理崩溃/归因竞态
- 退款 void 佣金

### M2：Partner Portal 前端（3-4 天）

- 独立 Vue app，路由：`/partners/*`
- 7 个页面：登录/注册/dashboard/链接/佣金/提现/KOL列表
- API 对接：调用 `/api/referrals/*` 端点

### M3：营销站改动（1-2 天）

- `?ref` cookie 写入逻辑
- 「成为推广伙伴」CTA
- 登录后跳转逻辑

### M4：合规（1-2 天）

- 推广协议文本
- 种草文案模板
- 免责声明补推广页
- 防刷规则实现

### M5：上线（2 天）

- Caddy 路由配置
- E2E 测试（归因/佣金/退款/防刷）
- 部署

**总计约 2 周**

---

## 十、决策记录

| 项 | 决定 | 理由 |
|---|---|---|
| 产品定价 | ¥88 | 起名服务合理价位，88=吉利 |
| 归因窗口 | 30 天 cookie | 首次点击后 30 天内注册归因；归因后的用户关系不再过期 |
| 佣金率 | KOL 50% / 招商 20% | 高于市场吸引人，v1 先跑通 |
| 提现门槛 | ¥50 / $10 | 国内支付宝、海外 PayPal，人工打款 |
| 伙伴门槛 | 付费用户即可开通 | 用过产品才能推广，¥88 筛选承诺 |
| 合作伙伴入口 | 邀约优先，v2 开放 | v1 验证转化率 |
| 路径 | `/partners/*` | 无子域，简洁 |
| 前端 | Vue（同主站） | 同技术栈，一人维护 |
| 产品 | 起名为主 | 海外华人刚需、低监管、高客单 |
| 裂变 | 不做 | 一层佣金（招商→KOL→用户），标准联盟营销 |
| 命理师培训/证书 | 不做（远期） | 先验证转化率 |
| 角色命名 | 待定 | 先不管，后续再定 |
