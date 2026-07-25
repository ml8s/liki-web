# 五行配色体系 — 全站色彩规范

## 色彩映射

| 五行 | 色名 | 色值 | 语义 | 用途 |
|------|------|------|------|------|
| 木（用神） | 竹青 | #5b8c5a | `--color-wood` | 主色 — 按钮/链接/焦点 |
| 木深 | 苍绿 | #3d6b3e | `--color-wood-hover` | hover/active |
| 木浅 | 青白 | #e8f0e0 | `--color-wood-soft` | 浅色背景/标签/徽章 |
| 火（喜神） | 朱砂 | #b85450 | `--color-fire` | 点缀 — 发送按钮/红点 |
| 火深 | 赭石 | #8c3b39 | `--color-fire-hover` | hover |
| 金 | 金色 | #e4c060 | `--color-gold` | 品牌 logo/header（缩小面积） |
| 金亮 | 浅金 | #e4c878 | `--color-gold-light` | header hover |
| 土 | 宣纸 | #fdfaf7 | `--color-paper` | 页面背景 |
| 土面 | 麻布 | #f3f0eb | `--color-surface` | 卡片/区块背景 |
| 土线 | 陶土 | #e2ddd6 | `--color-border` | 边框/分割线 |
| 水 | 墨玉 | #1e2532 | `--color-text` | 正文 |
| 水淡 | 烟灰 | #6c655f | `--color-text-muted` | 辅助文字/说明 |
| 水线 | 灰白 | #e7e5e4 | `--color-stone-100-raw` | 轻分割线 |

## 应用规则

### 全局 — Header
```
背景: header-dark (深色)
logo:  金色   --color-gold
文字:  浅金   --color-gold-light
按钮:  border --color-gold
```

### 全局 — 按钮（通用 .btn-primary）
```
背景:    竹青    --color-wood
hover:   苍绿    --color-wood-hover
文字:    白色    #fff
```

### 全局 — 链接
```
常态:    竹青    --color-wood
hover:   苍绿    --color-wood-hover
```

### 全局 — 输入框焦点
```
边框:    竹青半透 --color-wood 30%
阴影:    rgba(91,140,90,0.15)
```

### 产品页 — CTA 按钮（购买）
```
背景:    竹青    --color-wood
hover:   苍绿    --color-wood-hover
```

### Chat 页 — 发送按钮
```
背景:    朱砂    --color-fire
hover:   赭石    --color-fire-hover
```

### Chat 页 — 用户消息气泡
```
背景:    青白    --color-wood-soft
```

### 报告页 — 评级徽章
```
背景:    青白    --color-wood-soft
文字:    苍绿    --color-wood-hover
```

### 管理后台 — 暂不调整

## 不该用的颜色
```
❌ #2d6a4f（太鲜艳，与暖纸色打架） — 之前错误的值
❌ 纯红色（与木不合）
❌ 大面积金色（金克木）
```
