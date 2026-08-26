# 全站色调调整方案 — 五行配色

## 原则
- **不改 class 名** — `bg-amber-600` 继续叫 amber，颜色变绿
- **只改 CSS 变量值** — 8 行 primitive token + 3 行 semantic alias = 11 行
- **硬编码色值单独修** — 2 处 .brand-link + common.js

## 配色映射

| 五色 | 旧色 | 新色 | 用途 |
|------|------|------|------|
| 木 | — | 墨绿 #2d6a4f | **主色** — 按钮/链接/焦点 |
| 火 | 金 #c8963e | 暖橙 #d97706 | **点缀** — 发送按钮/高亮 |
| 金 | 金 #e4c060 | **缩小** — 仅限品牌 logo |
| 土 | 琥珀 #9c5e0b | **移除** — 被木取代 |
| 水 | 深 #1e2532 | **不变** — 文字 |

## 改动清单

### 1. app.css — primitive tokens（8 行）

| 变量 | 旧值 | 新值 | 行 |
|------|------|------|----|
| `--color-accent-raw` | #9c5e0b（琥珀） | **#2d6a4f**（墨绿） | 17 |
| `--color-accent-hover-raw` | #78350f | **#1b4332**（深绿） | 18 |
| `--color-accent-soft-raw` | #fef3c7 | **#d8f3dc**（浅绿） | 19 |
| `--color-fire-raw` | #c8963e（金） | **#d97706**（暖橙） | 23 |
| `--color-fire-hover-raw` | #d4a24e | **#b45309**（深橙） | 24 |
| `--color-gold-raw` | #e4c060 | **不变** | 25 |
| `--color-gold-light-raw` | #e4c878 | **不变** | 26 |

### 2. app.css — derived colors（2 行）

| 变量 | 旧值 | 新值 | 行 |
|------|------|------|----|
| `--color-accent-focus` | rgba(156,94,11,0.15) | **rgba(45,106,79,0.15)** | 54 |
| `--color-gold-alpha-20` | rgba(228,200,120,.2) | **不变** | 79 |

### 3. 删除 wood 临时变量（3 行）

`--color-wood-raw/hover-raw/soft-raw` 不再需要，因为已通过 `--color-accent` 使用。

### 4. 硬编码色值修复（2 处）

| 位置 | 旧 | 新 | 原因 |
|------|----|----|------|
| `web/css/app.css .brand-link` | `#e4b84c` | `var(--color-gold)` | 对齐 token 系统 |
| `web/js/common.js:192` | `color:#e4b84c` | `color:var(--color-gold)` | 对齐 token 系统 |

## 不改的
- 所有 `.astro` 文件（class 名不变，颜色自动变）
- PaymentRow.astro 的 `bg-amber-600 hover:bg-amber-700`
- LangSwitcher.astro 的 `bg-amber-50 text-amber-700`
- 所有命名类 `bg-amber-*` / `text-amber-*` 继续用原名

## 五行校验

| 色 | 调整后 | 五行 |
|----|--------|------|
| 墨绿 #2d6a4f 主色 | ✅ 按钮/链接/焦点框 | **木旺**（用神） |
| 暖橙 #d97706 点缀 | ✅ 发送按钮/CTA | **火助**（喜神） |
| 金 #e4c060 品牌 | ✅ 仅限 header logo | **金退**（不克木） |
| 白/灰背景 | ✅ 不变 | **金生水**（无害） |
| 深色文字 | ✅ 不变 | **水藏** |

## 实施步骤
```
1. app.css: 5 行 primitive token 换值
2. app.css: 1 行 --color-accent-focus 更新
3. app.css: 删除 --color-wood* 临时变量（不再需要）
4. app.css: .brand-link 硬编码 → var(--color-gold)
5. js/common.js: 硬编码 #e4b84c → var(--color-gold)
6. build + 验证
```
