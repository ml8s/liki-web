# Slogan 落地计划：懂命理，用 Liki

- 状态：**已实施**（og-image 按用户指示未动）
- 目标：让「懂命理，用 Liki」成为品牌记忆锚点，在关键触点**一致重复**，占领用户心智
- 范围：营销站（Astro，3 locale）+ 登录后 App（Vue）

## 已定决策（本轮拍板）
- en slogan：**For Chinese Metaphysics, use Liki.**
- 首页 H1 保留关键词，slogan 作独立标语行（不动 H1）
- header 常驻 tagline：仅桌面端显示
- `og-image.png`：**未改动**
- meta/SEO：未逐页追加（避免稀释唯一描述，可见触点已足够）

## 0. 核心原则

1. **一字不改**：slogan 在所有触点**字符串完全一致**（含标点）——重复才有心智，改写即稀释。
2. **关键触点重复，不刷屏**：入口 / 常驻导航 / 转化 / 分享 / 异常——每个"关键时刻"出现一次，不堆砌。
3. **与品牌字统一**：slogan 用品牌宋体（`font-brand`/`font-serif`），与现有 `Liki·命理` 字标同一视觉语言。
4. **不伤 SEO**：首页 H1 保留关键词（命理 Skill），slogan 作为醒目的品牌标语行，不替换 H1。

## 1. Slogan 多语言定稿

| locale | 文案 |
|---|---|
| zh-Hant | **懂命理，用 Liki** |
| zh-Hans | **懂命理，用 Liki** |
| en | **For Chinese Metaphysics, use Liki.**（推荐；SEO 备选：For Chinese metaphysics, use Liki.） |

- i18n key：`site.slogan`（单一来源，全站复用）
- 标点：中文用全角「，」；英文两句号。zh-Hant 用「Liki」（brand v4.0：中文亦用拉丁品牌名 Liki，不再用「靈機」）。

## 2. 触点矩阵（"重复"的落点）

| # | 触点 | 位置 | 形式 | 频率 |
|---|---|---|---|---|
| 1 | 营销站 · 首页 hero | H1 下方 | **大号宋体标语**（主曝光） | 每访 |
| 2 | 营销站 · header | 品牌字右侧 | 小号 tagline（常驻） | 每页 |
| 3 | 营销站 · footer | 版权上方 | 一行 slogan | 每页 |
| 4 | 营销站 · 产品页 CTA | 页面底部转化区 | slogan + 按钮（"懂命理，用 Liki" → 开始） | 每产品页 |
| 5 | 营销站 · 404 | 中央 | slogan + 返回首页 | 异常时 |
| 6 | 营销站 · meta/SEO | `<title>` / `og:description` / `twitter` | 每页 meta 尾缀 slogan | 每页 |
| 7 | 分享卡 | `og-image.png` | 图上叠加 slogan | 每次分享 |
| 8 | App · 登录页 | 品牌区 | slogan 作副标 | 每次登录 |
| 9 | App · Shell header | `Liki·命理` 下方 | 小号 tagline（常驻） | 每次使用 |
| 10 | App · 空态/加载 | 文案 | 语境化变体（如"还没有档案 · 懂命理，用 Liki"） | 按需 |

> 首页触点数：1+2+3+6+7 = **5 处**；其余每页至少 3 处（2+3+6）。既密集又不打扰。

## 3. 视觉与排版建议

- **首页 hero**：slogan 用 `font-brand`（宋体）、字号介于 H1 与副标之间（如 text-2xl~3xl）、字距略宽；作为独立一行，与 H1、副标形成"品牌 → 标语 → 说明"三段式。
- **header tagline**：`text-sm text-stone-400`，与品牌字同一行、垂直居中对齐，移动端隐藏（保清爽）。
- **CTA 区**：slogan 作标题，按钮沿用现有主按钮样式（品牌紫）。
- **404**：slogan 居中大号，下接"返回首页"。
- **一致性**：任何触点不得改写 slogan 措辞或标点。

## 4. 实施步骤

| 阶段 | 内容 | 涉及文件 |
|---|---|---|
| P0 | i18n 加 `site.slogan`（3 locale） | `web/i18n/{zh-Hant,zh-Hans,en}.json` |
| P1 | 营销站：hero / header / footer / CTA / 404 / meta | `src/pages/[locale]/index.astro`、`src/layouts/BaseLayout.astro`、`src/components/Footer.astro`、产品页 `*.astro`、`404.html` |
| P2 | SEO 与分享卡：meta 尾缀 + og-image 叠字 | `BaseLayout.astro`、`web/img/og-image.png`（重新生成） |
| P3 | App：登录页 / shell / 空态 | `web/app/src/views/LoginView.vue`、`AppLayout.vue`、`components/app/AppEmpty.vue` 等 |
| P4 | 校验 | `make check-html check-i18n`（i18n key 三语齐全） |

## 5. 验收标准

- 首页 slogan 出现 **≥5 处**（hero/header/footer/meta/og），其余营销页 **≥3 处**
- App 登录页与 shell 各 ≥1 处
- 三语言齐全，`make check-i18n` 通过
- slogan 字符串在所有触点**完全一致**（含标点）
- 不破坏 a11y（`make check-html`）与移动端布局

## 6. 开放项（需你定）

1. en slogan 采用 **"For Chinese Metaphysics, use Liki."**（推荐）还是 SEO 向的 "For Chinese metaphysics, use Liki."？
2. 首页 hero 是否把 slogan 提升为主标题（会牺牲部分 SEO 关键词）？——**建议不提升**，保留 H1 关键词。
3. header 常驻 tagline 是否要（会增加视觉密度）？——**建议要**，但仅桌面端显示。
4. 是否重做 `og-image.png`（叠 slogan）？——**建议要**。