# SEO & GEO 执行规划（liki-web）

> 受 [BRAND.md](./BRAND.md) 约束：品牌名、品类、Slogan 等一切品牌语言以 BRAND.md 为准。本文档按**页面（URL）**记录 title / H1 / H2 的最终文案与优化状态。
> 中英文各自独立规划，以各自市场的真实搜索数据为准（数据来源见 §7）。

## 1. 原则

1. **关键词池**：带 Chinese 与不带 Chinese 的搜索词同属一个池子（chinese horoscope 与 feng shui、i ching、wedding date calculator 并列）。
2. **意图匹配**：每个目标词与页面能力一一对应，避免泛词造成的意图错配（如泛指西方占星的 horoscope 不作门面目标词）。
3. **页面承接**：大词池由专门页面承接；title 放主轴词，次级词由 H2 与正文覆盖。
4. **词池迭代**：每轮热词调研后，先更新 §3 中**现有页面**的目标词与文案，再扩充 §4 机会池。
5. **场景体系**：营销页 ↔ App 入口按**用户价值场景 1:1**（七场景：看命 / 流年 / 合盘 / 问卦 / 择日 / 风水 / 起名）；内部分类（四专家部署）不呈现给用户；「免费 + 工具」是每个场景页的必现要素（免费词族见 §4 P1.1）。

## 2. 页面 SEO 规范（元素约束）

| 元素 | 权重 | 约束 |
| --- | --- | --- |
| `<title>` | 最高 | 全站唯一；主轴词靠前；「… · Liki」收尾；≤ 60 字符；不与 H1 逐字重复 |
| `<h1>` | 高 | 全页唯一；承载主词或与 title 互补的意图词；不用通用文案或纯品牌口号替代主词 |
| `<h2>` | 高 | 常规页 3–6 个、**hub 页（natal / huangli）6–10 个**；每个承载一个次级词族；使用搜索语言，不用通用 UX 标签 |
| `<h3>` / 卡片标题 | 中 | 承载长尾变体（如年份、组合词） |
| 首段 | 高 | H1 后 1–2 句，含主词与价值承诺 |
| `meta description` | CTR | 主词 + 次级词 + 行动词；≤ 155 字符；自然语句 |
| URL | 中 | 简短、含词；上线后不变更 |
| 图片 `alt` | 中 | 描述性、含词 |
| 内链锚文本 | 中高 | 指向页面以其主词作锚文本 |
| `canonical` / `hreflang` | 技术 | 每页自指 canonical；三语 alternate 保持构建输出 |
| 结构化数据 | 实体 | 按页型：`Service`（能力页）/ `FAQPage`（问答）/ `SoftwareApplication`（Skill 页）；实体级用 BRAND.md 主定义句 |

**页面类型与 schema 映射**（现状已实现）：

| 页型 | 页面 | schema |
| --- | --- | --- |
| 首页 | `/` | Organization + WebSite + ItemList ✓ |
| 能力页 / 场景页 | natal / divination / fengshui / huangli / naming（已实现）· hepan / liunian（WP8 新建） | Service + BreadcrumbList + FAQPage ✓ |
| Skill 页 | skills | SoftwareApplication ✓ |

**共用 H2 键**（能力页 7 个页面同时生效）：

| 键 | 区块 | 最终文案（en / zh-Hans·zh-Hant） |
| --- | --- | --- |
| `scenario.stepsTitle` | 步骤区 | `How to Start a Chinese Astrology Reading` / `如何开始你的命理咨询` |
| `scenario.questionsTitle` | 问题示例区 | 保持 `Questions you can ask` / `你可以这样问`（长尾由内容承载） |
| `scenario.faqTitle` | FAQ 区 | 保持 `Frequently asked questions` / `常见问题`（FAQPage schema 引用） |

## 3. 页面清单（逐页 SEO 信息与最终文案）

### 首页

**`/en/`（英文）**

- title：`Chinese Horoscope, BaZi & Feng Shui, Online · Liki` ✓
- H1：`For Astrology, use Liki.`（slogan，保留）
- H2（最终）：
  1. `Free Chinese Horoscope & BaZi Charting`（免费排盘区；替换 "Free charting"）
  2. `BaZi, ZiWei, Feng Shui, Naming & Divination`（服务区；替换 "Six metaphysics scenarios"）
  3. `Why Trust Liki: Engine-Computed & Verifiable Readings`（信任区；替换 "Why trust us"）
  4. `Ask the AI Astrologer`（AI 顾问叙事区；承接 ai astrologer 词族）
  5. `Start Your Chinese Horoscope in Four Steps`（步骤区；替换 "Start in four steps"）
- description：✓ 维持（已含 zodiac / feng shui / naming）
- 内链：服务卡 → 六场景与能力页（锚文本用各服务主词）✓ 已有
- 状态：title 已就位；H2 待执行

**`/zh-Hans/`、`/zh-Hant/`（中文）**

- title：`懂命理，用 Liki`（维持，已有搜索表现）
- H2（最终）：
  1. `免费排盘：八字、紫微、八宅`（替换「免费排盘」）
  2. `八字、紫微、风水、起名等六大服务`（替换「六大命理场景」）
  3. `引擎排盘，结论可验证`（替换「为什么可信」）
  4. `向 AI 命理师提问`（AI 顾问叙事区）
  5. `四步开始你的命理咨询`（替换「四步开始」）
- description：✓ 维持
- 内链：同英文版 ✓
- 状态：维持 title；H2 待执行

### 命理（natal）

**`/en/natal/`（英文）**——原 bazi / mingshu 两页合并为命理专家 hub（hepan 已恢复独立页）

- title：`Chinese Astrology Chart & Life Reading · Liki`
- H1：`Chinese Astrology: BaZi, Zi Wei Dou Shu & Life Reading`
- H2（最终，全面承载命理线词族）：
  1. `Chinese Astrology Chart: Four Pillars & Day Master`（八字排盘区）
  2. `Zi Wei Dou Shu (Purple Star Astrology)`（紫微区）
  3. `Life Reading: Career, Wealth & Relationships`（命书区）
  4. `Luck Pillars, Five Elements & Annual Forecast`（大运流年区）
  5. `Chinese Zodiac & Your Sign: Personality by Animal`（生肖区）
  6. `Birth Time & True Solar Time`（考时区："出生时间不准怎么办"→ CTA，非工具）
  7. 共用键（见 §2）
- description：`Online Chinese astrology: calculate your BaZi chart, explore Zi Wei Dou Shu, and read your life destiny — with the AI consultant.`
- 次级词：bazi calculator · purple star astrology calculator · **birth time rectification**（内容区块：出生时间不准怎么办）· true solar time · is bazi accurate（FAQ）
- 301 来源：`/en/bazi/` · `/en/mingshu/` → `/en/natal/`（锚点：`#bazi` / `#ziwei`）；**hepan 恢复独立页（WP8 撤销其 301）**
- 形态：**工具页 + 内容区块**——输入出生时间 → 免费工具出排盘（四柱/日主/五行）→ CTA 进 App 深度解读；下方内容区承载紫微/命书词族
- 内链：→ divination · fengshui · huangli
- 状态：已上线；合盘按 WP8 拆出独立页（hepan），本页词面去除合盘

**`/zh-Hans/natal/`、`/zh-Hant/natal/`（中文）**——原 bazi / mingshu 两页合并（hepan 已恢复独立页）

- title：`在线命理：八字排盘与紫微斗数 · Liki`
- H1：`在线命理：八字、紫微与命书`
- H2（最终）：
  1. `八字排盘：四柱、日主与十神`
  2. `紫微斗数：命宫、事业与财运`
  3. `命书：性格、事业、财富与姻缘`
  4. `大运流年与五行喜忌`
  5. `生肖属相：十二生肖性格`
  6. `出生时辰与真太阳时`
  7. 共用键（见 §2）
- 次级词：—（属相婚配等合盘词移至 hepan）
- 301 来源：`/zh-Hans/bazi/` · `/mingshu/`（zh-Hant 同）→ `/{locale}/natal/`；**hepan 恢复独立页（WP8）**
- 状态：已上线；合盘按 WP8 拆出（hepan）

### 合盘（hepan）——WP8 恢复独立页（撤销 M2 的 301）

**`/en/hepan/`（英文）**

- title：`BaZi Compatibility & Marriage Matching · Liki`
- H1：`BaZi Compatibility: Marriage Matching & Relationship Reading`
- H2（最终）：
  1. `Compatibility Chart: Day Masters & Branch Relations`（合盘盘面区）
  2. `Marriage Matching & Timing`（合婚/时机区）
  3. `Chinese Zodiac Compatibility`（生肖配对区）
  4. `How to Read a Compatibility Chart`（内容区）
  5. 共用键（见 §2）
- description：`Free BaZi compatibility chart: match two birth charts for marriage and relationships — day master, branch relations and timing, with the AI consultant.`
- 次级词：bazi marriage matching · chinese zodiac compatibility · love compatibility · marriage timing · soulmate
- 形态：**工具入口页**——一期 CTA 直达 App 合盘（`/app/synastry`）；二期可补合盘岛（与其余工具页同构）
- 工具区标题：`Free BaZi Compatibility Chart — Marriage Matching`
- 内链：→ natal（个人命盘）· → divination
- 状态：WP8 新建（内容从 natal 合盘区块迁移）

**`/zh-Hans/hepan/`、`/zh-Hant/hepan/`（中文）**

- title：`在线合盘：八字合婚与姻缘配对 · Liki`
- H1：`八字合盘：合婚、姻缘与关系解读`
- H2（最终）：
  1. `合盘盘面：日主与地支关系`
  2. `合婚与姻缘：正缘与结婚时机`
  3. `生肖配对与婚配`
  4. `怎么看合盘`
  5. 共用键（见 §2）
- description：`免费八字合盘：输入两人出生时间，看日主、地支关系与姻缘配对，由 AI 命理师做关系解读。`
- 次级词：八字合婚 · 姻缘 · 正缘 · 生肖配对 · 属相婚配
- 工具区标题：`免费合盘｜八字合婚 · 姻缘配对`
- 301：**撤销**（恢复独立 URL，不再指向 natal）

### 流年（liunian）——WP8 新建

**`/en/liunian/`（英文）**

- title：`Chinese Annual Fortune: Yearly Horoscope · Liki`
- H1：`Annual Fortune: Your Year Ahead in BaZi`
- H2（最终，年度内容逐年更新）：
  1. `Your Annual Chart: Year Pillar & Ten Gods`（流年盘区）
  2. `Tai Sui & Ben Ming Nian (2026)`（犯太岁区，逐年更新）
  3. `Annual Relations with Your Chart & Luck Pillars`（关系区）
  4. `How to Read Your Year Ahead`（内容区）
  5. 共用键（见 §2）
- description：`Free annual fortune chart: your year pillar, yearly relations and Tai Sui for 2026 — with an AI astrologer reading of your year ahead.`
- 次级词：2026 chinese horoscope · annual fortune · yearly forecast · tai sui · ben ming nian
- 形态：**工具页**——年份选择 → 免费流年盘（引擎 `bazi_liunian`，复用 App 流年组件）→ CTA 年度解读
- 工具区标题：`Free Annual Fortune Chart — 2026 Chinese Horoscope`
- 内链：← natal（命局）· → huangli（每日/择日）
- 状态：WP8 新建

**`/zh-Hans/liunian/`、`/zh-Hant/liunian/`（中文）**

- title：`流年运势：年运程与犯太岁 · Liki`
- H1：`流年运势：这一年，你的机会与注意`
- H2（最终，年度内容逐年更新）：
  1. `流年盘：流年干支与十神`
  2. `犯太岁与本命年（2026 丙午）`
  3. `流年与命局、大运的关系`
  4. `怎么看流年`
  5. 共用键（见 §2）
- description：`免费流年运势：看流年干支、十神与犯太岁，结合命局大运，由 AI 命理师解读你的这一年。`
- 次级词：流年运势 · 2026年运势 · 犯太岁 · 本命年 · 年运程
- 工具区标题：`免费流年运势｜年运程与犯太岁`
- 内链：← natal · → huangli（每日黄历/择日）
- 状态：WP8 新建

### 问卦（divination）

**`/en/divination/`（英文）**

- title：`Online Fortune Telling: Ask a Question · Liki` ✓
- H1：`I Ching Reading & Liu Yao Divination Online`（替换 "Liu Yao divination and timing"）
- H2（最终）：
  1. `I Ching Reading: Hexagram, Useful God & Timing`（能力区）
  2. `Liu Yao Reading Example`（示例区）
  3. 共用键（见 §2）
- description：微调补 I Ching —— `Online fortune telling with I Ching (Liu Yao): ask one question and read the hexagram, useful god, changing lines and timing — interpreted by the AI consultant.`
- 内链：→ natal（Chinese Astrology）
- 状态：title 已就位；H1/H2 待执行

**`/zh-Hans/divination/`、`/zh-Hant/divination/`（中文）**

- title：`六爻问卦与应期解读 · Liki`（维持）
- H2（最终）：
  1. `六爻占卜：卦象、用神与应期`（替换「Liki Chat 问卦可以看什么」）
  2. `在线占卜示例`（替换「问卦解读结构示例」）
  3. 共用键（见 §2）
- description：微调补「在线」—— `在线六爻问卦解读：聚焦具体问题，理解卦象、用神、变爻与应期，并获得更清晰的行动参考。`
- 内链：→ natal（八字排盘）
- 状态：维持 title；H2 待执行

### 风水（fengshui）

**`/en/fengshui/`（英文）**

- title：`Feng Shui Bedroom & Home Layout Guide · Liki` ✓
- H1：`Feng Shui for Bedroom, Home Layout & Directions`（替换 "Feng Shui directions and home reading"）
- H2（最终）：
  1. `Feng Shui Bedroom, Directions & Bagua Map`（能力区）
  2. `Home Feng Shui Reading Example`（示例区）
  3. 共用键（见 §2）
- description：✓ 维持（bedroom / directions 已覆盖）
- 内链：→ huangli（Date Selection）
- 状态：title 已就位；H1/H2 待执行

**`/zh-Hans/fengshui/`、`/zh-Hant/fengshui/`（中文）**

- title：`风水方位与宅盘解读 · Liki`（维持）
- H2（最终）：
  1. `家居风水：卧室、方位与八宅`（替换「Liki Chat 风水解读可以看什么」）
  2. `宅盘解读示例：八宅与飞星`（替换「风水解读结构示例」）
  3. 共用键（见 §2）
- description：✓ 维持
- 内链：→ huangli（黄历择日）
- 状态：维持 title；H2 待执行

### 择日 / 黄历（huangli）

**`/en/huangli/`（英文）**

- title：`Wedding Date Calculator & Auspicious Dates · Liki` ✓
- H1：`Chinese Almanac & Auspicious Date Selection`（替换 "Chinese calendar and date selection"）
- H2（最终）：
  1. `Chinese Almanac: Auspicious & Inauspicious Days`（能力区）
  2. `Wedding & Moving Date Examples`（示例区）
  3. `Year of the Horse 2026 & Tai Sui`（流年太岁区；**逐年更新区块**）
  4. 共用键（见 §2）
- 次级词：chinese wedding date calculator · auspicious wedding dates 2026 · lunar calendar converter · **chinese lunar birthday** · **chinese almanac today（每日区块）**
- description：✓ 维持
- 内链：→ fengshui（Home Layout）
- 状态：title 已就位；H1/H2 待执行

**`/zh-Hans/huangli/`、`/zh-Hant/huangli/`（中文）**

- title：`黄历择日与宜忌解读 · Liki`（维持）
- H2（最终）：
  1. `黄历宜忌：吉日与不宜`（替换「Liki Chat 择日可以看什么」）
  2. `择日示例：嫁娶、开业与搬家`（替换「择日解读结构示例」）
  3. `2026 马年运程与犯太岁`（流年太岁区；**逐年更新区块**）
  4. 共用键（见 §2）
- 次级词：今日黄历（每日区块）· 农历生日查询
- description：✓ 维持
- 内链：→ fengshui（风水布局）
- 状态：维持 title；H2 待执行

### 起名（naming）

**`/en/naming/`（英文）**

- title：`Chinese Baby Names & Name Analysis · Liki` ✓
- 形态：**工具页 + 内容区**——输入出生时间 → 免费算用神/喜忌（`bazi_yongshen`）→ 起名用神方向（确定性输出，不生成名字）→ CTA 进 App（AI 命名）
- H1：`Chinese Baby Names & Name Analysis`（替换 "AI naming and name interpretation"）
- H2（最终）：
  1. `Chinese Baby Names & Five-Element Naming`（能力区）
  2. `Chinese Name Examples & Name Generator`（示例区）
  3. 共用键（见 §2）
- 次级词：what is my chinese name · chinese name meaning
- description：✓ 维持
- 内链：→ natal（命理排盘，起名结合八字）
- 状态：title 已就位；H1/H2 待执行

**`/zh-Hans/naming/`、`/zh-Hant/naming/`（中文）**

- title：`AI 起名 · Liki`（维持）
- 形态：**工具页 + 内容区**——输入出生时间 → 免费算**用神/喜忌**（`bazi_yongshen`）→ 展示「起名用神方向 + 命名原则」（确定性输出，**不生成名字**）→ CTA 进 App：AI 命名（LLM 质量把关 + naming 工具链）
- H2（最终）：
  1. `宝宝起名：五行与喜用神`（替换「Liki Chat 可以怎样帮你起名」）
  2. `起名示例与姓名测试`（替换「名字思路示例」）
  3. 共用键（见 §2）
- 次级词：男宝宝取名 · 女宝宝取名（细分区块）
- description：✓ 维持
- 内链：→ natal（八字排盘）
- 状态：维持 title；H2 待执行

### Skill 页（skills）

**`/en/skills/`（英文）**

- title：`Professional Astrology Skill · Liki`（待从 `Professional Skill for Chinese Metaphysics` 同步）
- H1：`Liki Skill`（产品名式，替换 `Liki — Professional Skill for Chinese Metaphysics`）
- H2（新增，当前整页无 H2）：
  1. `How to Install Liki Skill`（安装区，覆盖 `npx skills add`）
  2. `What Liki Skill Covers`（能力区）
- description：待同步（随品牌批次改用 `Professional Astrology Skill` 表述）
- 内链：→ llms.txt · GitHub ✓ 已有
- 状态：待执行（随品牌同步批次）

**`/zh-Hans/skills/`、`/zh-Hant/skills/`（中文）**

- title：`命理 Skill · Liki`（保留，已有 Bing 搜索表现）
- H1：`Liki Skill`（产品名式，替换 `Liki — 专业命理 Skill`）
- H2（新增）：
  1. `安装 Liki Skill`
  2. `能力覆盖`
- description：✓ 维持
- 内链：→ llms.txt · GitHub ✓ 已有
- 状态：待执行

### 页面蓝图与组件（实现规格）

五类页面共用一套骨架（沿用营销站现有类），差异在「工具岛 + 区块组」：

**统一骨架**（专家页 / huangli / 首页同构）：

| # | 区块 | 内容 | 组件 | 布局（现有类） |
| --- | --- | --- | --- | --- |
| 1 | Hero | H1 + 副题 + 主 CTA | `BaseLayout`（landing-hero） | 全宽，窄内容 |
| 2 | **工具岛** | 输入 → 免费结果（确定性输出）→ CTA | Vue island（`client:load`） | 窄卡（约 max-w-3xl） |
| 3 | 能力区块组 | H2 ×N（各自词族；含示例/步骤） | 静态 Astro（扩展复用 `ScenarioLanding` 结构） | landing-services 网格 |
| 4 | 步骤区 | 4 步开始 | 现有 steps 区 | landing-steps |
| 5 | FAQ | details 折叠（含信任问题） | 现有 FAQ 区 | 窄列 |
| 6 | 底部 CTA | 进 App | `LikiChatCta` | landing-cta |

**每页差异**：

| 页面 | 工具岛（新增 Vue 岛） | 区块组（H2 按 §3） |
| --- | --- | --- |
| natal | `NatalTool`：出生输入（`BirthInputForm`）→ `PaipanResult`（含紫微/合盘结果）→ CTA | 八字 · 紫微 · 合盘姻缘 · 命书 · 流年 · 生肖 · 考时 |
| naming | `NamingTool`：出生输入 → `YongshenPanel`（用神方向，**不生成名字**）→ CTA | 宝宝起名 · 五行取名 · 姓名测试 · 男女宝细分 |
| divination | `GuaTool`：问题输入 → `GuaPanel`（卦象）→ CTA | I Ching · 六爻 · 在线占卜 · 示例 |
| fengshui | `FengshuiTool`：出生（+朝向）→ `BazhaiPanel`（命卦/方位）→ CTA | 家居风水 · 方位 · 八宅/bagua |
| huangli | `HuangliTool`：日期 + 事件 → `AuspiciousDays`（吉日列表）→ CTA | 黄历宜忌 · 择日 · 流年太岁 · 每日 · 历法 |
| 首页 | `paipan-hook` 升级为 `PaipanTool`（精简版 Natal 岛） | 现有区块 + AI 顾问区 |
| skills | 无工具岛（安装区现成） | 现有 + How to Install / What It Covers |

**组件清单**：

| 层 | 组件 | 说明 |
| --- | --- | --- |
| 共享 API | `web/shared/api/engine.ts` | engine tool 调用封装 + 类型（营销 / App 共用） |
| 共享展示（Vue） | `BirthInputForm` · `PaipanResult` · `ZiweiPanel` · `GuaPanel` · `AuspiciousDays` · `BazhaiPanel` · `YongshenPanel` | 纯展示 / 回调驱动、无 store；App 侧同步替换内联实现 |
| 营销岛（Vue） | `NatalTool` · `NamingTool` · `GuaTool` · `FengshuiTool` · `HuangliTool` | 组合「输入 + API + 展示 + CTA」；Astro 以 `client:load` 挂载 |
| 营销静态（Astro） | `ScenarioLanding`（扩展为通用区块组）· `LikiChatCta` · `BaseLayout` | 现有件复用 |

**布局与容器**：沿用现行营销容器（主列 768px；landing-* 区块节奏与间距 token）；工具岛与区块组共用该容器，不改全站布局；hub 页区块数量更多，维持既有区块间距。

## 4. 词嵌入清单（机会词 → 归属页 → 区块）

**原则：零新页**——所有机会词嵌入七页体系（首页 / natal / naming / divination / fengshui / huangli / skills）的区块；年度内容为常青页上的**逐年更新区块**（不新建 URL、不废弃页面）。

### P1 · 高优先嵌入

| 词族 | 归属页 | 区块（新增 / 已有） |
| --- | --- | --- |
| 生肖 · chinese zodiac personality · what is my chinese zodiac | natal | 生肖区块（新增 H2） |
| 八字 / four pillars / day master / luck pillars · 紫微 / purple star astrology · 命书 / life reading | natal | 已有 H2 覆盖 |
| 姻缘 · soulmate · marriage timing · 正缘 · 八字合婚 · 属相婚配 | **hepan** | 合盘页（H2 + 姻缘区块） |
| 五行 · five elements calculator / personality | natal | 五行区块（随工具结果） |
| 真太阳时 · 考时 · birth time rectification | natal | 考时区块（内容 + CTA；需 AI，非公开工具） |
| 幸运数字 / 颜色 · chinese lucky numbers | natal | 幸运元素区块（随工具结果） |
| 2026 马年 · 犯太岁 · 本命年 · 拜太岁 · 2026 运势 · 流年运程 | **liunian** | 流年页（流年盘 + 犯太岁区块，逐年更新；huangli 现有太岁内容块保留互链） |
| 今日黄历 · chinese almanac today | huangli | 每日宜忌区块（引擎数据，可自动化） |
| 农历生日 · lunar birthday · 农历转换 | huangli | 历法区块 |
| 婚礼 / 开业 / 搬家 / 剖腹产择日 · wedding date · auspicious dates | huangli | 择日区块（事件类型扩列） |
| 男宝 / 女宝起名 · chinese name for boy / girl · 公司起名 | naming | 起名细分区块 |
| ai astrologer · best ai for astrology · ai horoscope | 首页 | AI 顾问叙事区块 |
| i ching · 六爻 · 在线占卜 · 算卦 | divination | 工具 + 已有 H2 强化 |
| feng shui bedroom / layout · bagua · 八宅 | fengshui | 已有 H2 强化 |

### P1.1 · 工具词族（M4 工具区块词嵌入；2026-10 调研）

**原则：工具区块本身就是最强的词承载体**——搜索者要的就是"这个免费工具"；工具区 H2/引言按下列词族定稿（内容区块承载解释性长尾，工具区承载交易型短词）。

| 工具（页） | 中文词族 | 英文词族（竞品共识，已交叉验证） |
| --- | --- | --- |
| 八字盘（natal） | 免费八字排盘 · 八字排盘 · 在线排盘 · 四柱八字 · 生辰八字 · 十神 · 藏干 · 大运 | free bazi calculator · bazi chart · four pillars of destiny · day master · luck pillars · free four pillars chart |
| 紫微盘（natal） | 紫微斗数在线排盘 · 免费紫微斗数 · 十二宫 · 命宫 | zi wei dou shu chart · purple star astrology · twelve palaces · free zi wei chart |
| 真太阳时（natal） | 真太阳时 · 出生时间校正 | true solar time · birth time rectification · unknown birth time |
| 起卦（divination） | 六爻在线排盘 · 在线起卦 · 摇卦 · 易经占卜 · 算卦 · 本卦变卦 | i ching online · cast a hexagram · free i ching reading · 64 hexagrams · three coins / yarrow stalks |
| 择日（huangli） | 老黄历 · 今日黄历 · 择吉 · 吉日查询 · 嫁娶吉日 · 开业吉日 · 搬家吉日 · 建除十二神 · 宜忌 | chinese almanac today · tong shu · auspicious dates · wedding date calculator · twelve day officers · yi and ji |
| 命卦（fengshui） | 命卦查询 · 八宅 · 东四命西四命 · 吉凶方位 · kua number | bagua · life gua · kua number · eight mansions · feng shui directions |
| 用神（naming） | 八字起名 · 五行取名 · 喜用神 · 宝宝起名 | bazi naming · favorable elements · five elements naming · chinese name by bazi |
| 合盘（hepan） | 免费合盘 · 八字合婚 · 合婚测试 · 姻缘配对 · 生肖婚配 | free compatibility chart · bazi marriage matching · chinese zodiac compatibility · love compatibility |
| 流年（liunian） | 流年运势 · 2026年运势 · 免费年运程 · 犯太岁查询 · 本命年 | annual fortune · 2026 chinese horoscope · yearly forecast · tai sui |
| Liki Skill（skills） | （已有覆盖） | mcp server · ai astrology mcp |

**竞品共识要点（供文案与功能验证）**：①"free / 免费 + no sign-up / 无需注册"是工具区必现词（几乎所有头部工具都强调）；②英文侧 "true solar time" 与 "unknown birth time" 是高频信任点；③黄历侧 "tong shu / 12 day officers / clash" 是差异化词；④工具区同时给"确定性计算、非 AI 猜测"的承诺（我们架构天然符合：引擎计算 + AI 解读分离）。

### P2 · 观察（数据验证后嵌入）

- Qi Men Dun Jia（奇门）→ divination 区块
- Chinese vs Western Astrology 对比内容 → natal 内容区块（合参叙事）
- Five Elements Personality Test 独立形态 → natal 五行区块增强
- Chinese Gender Calendar（清宫表）→ **暂缓**（产品形态未决）

## 5. GEO（生成引擎优化）

- **三形态陈述**：`llms.txt` 首段按 BRAND.md 的 Delivery 陈述（网页 AI 顾问 / Skill / MCP 服务）；实体级使用主定义句。
- **结构化数据**：只陈述事实；按 §2 规范选择类型。
- **稳定资产**：`/llms.txt`、`/skills/liki/SKILL.md`、`/.well-known/agent-skills/index.json`。
- **AI 组合词**由专门落地页承接（见 §4）。

## 6. 执行计划（工作包）

**总目标**：**七场景页**（看命 / 流年 / 合盘 / 问卦 / 择日 / 风水 / 起名）+ 工具区成为搜索入口；英文市场以 Chinese Astrology / AI Astrologer 词族收量；GEO 资产稳定。
**衡量**：新面收录 ≤ 4 周；目标词进前 20 后按 CTR 迭代；工具页 CTA 进 App 转化可追踪。
**量化目标（首期）**：英文面展示量 ≥ 当前 5×（当前约 73/月）；natal 页进入 chinese astrology / chinese horoscope 前 30；工具页 CWV 不劣于营销页基线。

### WP1 · 品牌与文案同步（✅ 已完成）

按 BRAND.md 的**面**逐处区分，不做全站统一替换：

- **能力面（AI 生态）**：`Professional Skill for Chinese Metaphysics` → `Professional Astrology Skill`——仅 skills 页 meta 与 liki 库 README.en.md。
- **消费者面**：残留该旧表述的页面 meta（about / disclaimer 等）→ 改用**主定义句派生**（`Professional AI Astrologer`），与首页同面。
- skills 页：H1 → `Liki Skill`；新增 2 个 H2（How to Install / What It Covers）；title 同步。
- e2e 断言同步（pages.spec 的 skills 标题）。
- 验收：lint / check-i18n / e2e 全绿；逐处核对表述与 BRAND.md 的面归属一致。

### WP2 · 结构重构：natal 合并 + 301（✅ 已完成）

- **迁移前基线盘点**：用 `seo page` / URL Inspection 记录 bazi / mingshu / hepan 的 GSC / Bing 收录与展示现状（301 决策与效果对比的依据）
- 新建 natal 页（三语）：内容 = 八字 + 紫微 + 合盘 + 命书 + 生肖 + 考时（§3 文案）；**结构与文案一次到位**（并入 WP5 相关项，避免二次改版）
- 旧页 301：`/{locale}/bazi|mingshu|hepan/` → `/{locale}/natal/`（Go fileServer 与 Caddy 分工实现时定）
- Caddyfile（dev + 生产）：裸路径 `/bazi` `/mingshu` `/hepan` 协商 → `/{lang}/natal`
- 首页服务卡与各页内链改指 natal；sitemap 自动更新
- 验收：301 生效、三语渲染、e2e 更新后全绿、GSC 收录转移监控

### WP3 · 工具化（核心）

- 3.1 共享层：`web/shared/api/engine.ts`（engine tool 调用 + 类型）；`web/shared/components/`（纯展示组件）
- 3.2 Astro 接入 Vue（@astrojs/vue）+ island 模式；先抽 `PaipanResult` 验证再推广
- 3.3–3.6 四工具 + huangli：natal（bazi_chart/fullchart）· divination（liuyao_qigua）· fengshui（bazhai_chart）· huangli（huangli_days，含每日区块）
- 3.7 naming 免费层：BFF 暴露 `bazi_yongshen`（counsel）→ 用神方向（**不生成名字**——公开工具层边界原则）
- 3.8 App 侧改造：使用共享组件（解耦，替换内联实现）
- **上线前**：引擎容量与成本评估（匿名工具 API 的预期调用量 vs 引擎容量）
- 验收：匿名出结果、限流（30/min）在线、island 体积在性能预算内、每工具 e2e 覆盖
- **执行记录（3.2 组件复用修订）**：排盘复用粒度 = **三个独立盘组件**（不搬整个一屏三盘）：`BaziPanel`（四柱/藏干/神煞/大运/用神）、`ZiweiPanel`（十二宫/格局/三方/空劫/大运）、`MingGuaSection`+`LiuNianFlySection`（八宅）——统一置于 `web/shared/components/paipan/`，纯展示、零 Element Plus 依赖（App 编排器的外壳/流年会话留在 App）。App 的 PaipanPreview 改为组装这三块的编排器（外观与行为零变化，engine-contract e2e 5 用例验证）。共享数据链新增 `baziPanFor`/`ziweiPanFor`/`bazhaiPanFor`（真太阳时/农历/校正提示统一）；排盘文案子树共享（`web/shared/i18n/chart-messages.<locale>.json` + 生成/同步校验脚本；后于深挖轮按 locale 拆文件）。**natal 岛已接入真实排盘**：表单 → 八字盘 + 紫微盘（e2e 断言四柱/大运/用神/十二宫）。
- **执行记录（3.5–3.6 ✅，原则：工具盘面一律复用 App 组件，一处优化全站生效）**：`fengshui` 岛（出生年+性别 → `bazhaiPanFor` → 共享 `MingGuaSection`+`LiuNianFlySection`）与 `huangli` 岛（事项+日期范围 → 共享 `engineHuangliDays` → 新抽共享 `HuangliDaysPanel`，App 的 HuangliView 同步改用；`EVENT_KEYS` 外置 `shared/utils/huangli.ts` 单一来源）均已上线；`ScenarioLanding` 新增 `tool` 槽；岛挂载抽为通用 `mountIsland`（island-boot.ts）；工具表单/结果骨架类（`.tool-*`）收敛到营销 CSS；共享消息扩至 `services.huangli` 子树；通用文案键 `tool.{submit,computing,error,cta,privacy}` 统一（natal 一并迁移）。e2e 新增 fengshui-tool/huangli-tool 两 spec；全组 75 passed（含 App engine-contract）。- **执行记录（3.4 ✅）**：`divination` 岛（免费起卦）已上线——严守公开工具层边界：匿名只调 `liuyao_qigua`（解卦 `liuyao_chart` 属 App 侧）；「本卦/变卦」由前端从 casting 推演（`shared/utils/liuyao.ts`，64 卦表由 engine `hexagrams.json` 生成、含自检），`LiuyaoCastPanel` 共享展示六爻爻线与动爻；e2e 断言「火泽睽 → 火水未济」推演正确。**M3 工具岛四件套（natal/divination/fengshui/huangli）完成。**
- - **执行记录（WP3 架构收口：BFF 引擎代理删除，前端直连 MCP）**：按"MCP 是统一对外服务"原则，删除了 liki-web 的 `internal/enginemcp`（客户端/白名单/自定义信封 ~700 行 + 装配/readiness/测试）——浏览器与外部 skill 现在走**同一路径与同一协议**（`/engine/mcp`、`/counsel/mcp`，网关剥前缀；两服务 stateless + JSONResponse，无需握手）。共享层 `engineTool` 直发 MCP JSON-RPC（Accept 头 + content 解包 + isError→ApiError；counsel 工具按名路由）；App/四岛/e2e 全部切换；dev Caddy 与生产（edge-owned 前缀早已就位）配置对齐；`web` 服务不再需要 MCP env。验证：Go 全量测试绿、App 110 单测、**e2e 81 passed**（含 App engine-contract 真契约渲染）。counsel 的 `bazi_yongshen` 由此直达（起名链路主用）。
- **执行记录（3.7 ✅，M3 完成）**：`naming` 岛（免费用神查询）上线——`bazi_chart`（engine）→ **`bazi_yongshen`（counsel，前端直连）** → 共享 `YongShenPanel`（用神/喜神/忌神 + 判据 + 低置信度/冲突复核提示）；只给方向、不生成名字（公开工具层边界）。共享消息扩至 `services.naming` 子树。e2e +3（用神渲染/复核提示/counsel 错误态）。
- **M3 收官**：工具岛五件套（natal/divination/fengshui/huangli/naming）全部上线，盘面与裁决组件全部与 App 同源、数据全部经同一 MCP 直连协议。剩余只待上线前：**引擎容量评估**（M5 前）。
- **执行记录（3.1–3.2 ✅）**：共享层已落 `web/shared/api/{http,engine}.ts`（ApiError + ENGINE_TOOLS + 全量引擎类型；App 侧 re-export、零调用点改动，vue-tsc/108 tests 全绿）。**展示组件策略修订**：营销岛按精简形态新写（App 的 PaipanPreview 与 BirthProfile 耦合深），共享边界 = API + 类型，展示组件按实际重复度再抽。island 挂载采用 **SSR 直出 + 外部 TS 脚本 createSSRApp hydration**（`data-props` 传参）——避开 Astro 的 inline island loader，维持 CSP 零 inline 可执行脚本（check-inline-scripts --dist 验证）。**natal 工具岛已上线**（3.3 的 natal 部分提前完成）：出生信息 → bazi_chart → 四柱/生肖/大运 → App CTA；e2e 覆盖成功/失败两态。剩余工具岛（divination/fengshui/huangli/naming）随 M3。

### WP4 · 词嵌入执行（零新页）

- natal 补区块：生肖 · 五行（随工具结果）· 考时 · 幸运元素
- huangli 补区块：流年太岁（2026 马年，逐年更新）· 每日宜忌 · 历法（农历生日/转换）· 择日事件扩列
- naming 补区块：男宝/女宝 · 公司起名等细分
- 首页补区块：AI 顾问叙事（ai astrologer 词族）
- 内容资产清单：上述区块所需的视觉与文案资源（生肖/年度区块为重点）
- 验收：三语文案、schema、内链、sitemap、**翻译质量抽查**（英文 native 语感、繁体本地化）

### WP5 · 逐页文案执行（§3 定稿落地；与 WP2 协同）

- H1 替换（divination / fengshui / huangli / naming / skills 共 5 处）；natal 页为新建（随 WP2 一次到位）；首页不动
- H2 文案（各页 + 共用键 3 个）· 首页 H2 词族化（含 AI 顾问区块）
- paipan-hook 升级（直接出盘，复用 WP3.3 组件）
- 验收：渲染文案与 §3 一致（可加 checks 脚本校验 title/H1 主词）；翻译质量抽查

- **执行记录（WP4 ✅）**：①**工具词族调研**（中英双线，交叉验证竞品）→ §P1.1；②**工具区 H2/intro 按搜索词重写**（natal/divination/fengshui/huangli/naming × 三语，承载 free bazi calculator / i ching online / life gua / date selection / yong shen 等交易词）；③**内容区块**：natal 工具结果新增五行统计与幸运元素（新 `shared/utils/wuxing.ts`），huangli 新增「流年太岁 2026 · 每日黄历 · 农历生日与转换 · 择日事件全覆盖」四区块（article 语义），naming 新增「男宝/女宝/公司起名」三分区；④首页 AI 顾问区块（M1 已就位）。
- **执行记录（WP6 ✅）**：`/llms.txt` 创建——首段按 BRAND.md 三形态陈述（网页 AI 顾问 / Skill / MCP 服务），实体级使用主定义句；免费工具与资产链接齐备。schema 已就绪（M2/M3 各页 Service/Breadcrumb/FAQ）。**M4 完成**。

### WP6 · GEO

- `llms.txt` 首段按三形态更新（网页 AI 顾问 / Skill / MCP 服务）
- schema 保持（已就绪）；实体主定义句统一

### WP7 · 上线与复盘

- **执行记录（容量评估 ✅，2026-10-19 生产实测）**：对 **liki.hk 生产链路**（Cloudflare → Caddy → MCP）直接压测五类代表工具（huangli 7d / bazi_chart / bazi_fullchart / ziwei / counsel bazi_yongshen），并发梯度 1→128、全程零错误。**结论**：
  - **服务端纯计算**（本地 loopback）：1–25ms/次（最重 huangli 7d 22.6ms、bazi_yongshen 25.4ms）——单核即可承载数十 QPS，计算远非瓶颈。
  - **端到端 p50 ≈ 0.85–1.2s**（跨洋测试路径）：99% 是网络/CF 往返；生产链路在 c=128 仍无错误、无延迟崩塌，~25–30 QPS 处出现链路饱和（非引擎饱和）。
  - **正常 SEO 流量**（工具页转化 ~1%）距饱和有余量数个数量级 → **无需扩容、无需复杂限流**。
  - **可选加固（按需，非现在）**：①CF 层对 `/engine/mcp`、`/counsel/mcp` POST 加速率规则（正常用户无感）；②若某工具成爆款再上"确定性结果缓存"（相同输入同输出，天然可缓存）；③CF 已提供最外层 bot 防护（非浏览器 UA 直接 403 实测确认）。


- 全链路验证（lint / typecheck / test / e2e / build / check-web-static）
- 部署：liki-web 镜像 + liki-deploy Caddy（含新 301 与协商规则）
- 上线后：`crawl` 复扫 · GSC / Bing URL Inspection · **GSC 与 Bing WMT 双端提交 sitemap 并请求重新抓取**
- T+2–4 周：`report` 复盘（目标词展示 / CTR）→ 迭代

### WP8 · 场景补齐：合盘拆页 + 流年新页（2026-10 决定）

- **合盘（hepan）恢复独立页**：迁 natal 的合盘区块与 FAQ 到新页；撤销 `/{locale}/hepan/` 的 301（Caddyfile + Go legacyPageTargets）；natal 词面去除合盘
- **流年（liunian）新页**：年份选择 → 免费流年盘（引擎 `bazi_liunian`，复用 App 流年组件）→ 年度解读 CTA；犯太岁区块（逐年更新）
- **免费词**：两页工具区标题按 §4 P1.1 落地（免费合盘 / 免费流年运势）
- **App 侧**：七场景定义单一来源（含图标字：命 / 运 / 合 / 卦 / 吉 / 名 / 宅）；入口统一（行式组件）；四专家隐去（会话标题/列表按场景）
- 验收：七页 ↔ 七入口 1:1；e2e（含隔离守卫）全绿；GSC 收录监控

### 里程碑与顺序

| 里程碑 | 内容 | 依赖 |
| --- | --- | --- |
| M1 | WP1 + WP5（文案与结构先行，低风险）✅ | — |
| M2 | WP2 + WP3.1–3.2（natal 合并 + 共享层）✅ | M1 |
| M3 | WP3.3–3.8（工具化全量） | M2 |
| M4 | WP4（词嵌入）+ WP6 ✅ | M3（复用组件与模式） |
| M4.5 | WP8（场景补齐：合盘/流年 + 免费词） | M4 |
| M5 | WP7（上线与复盘） | M4.5 |

### 风险与边界

- **301 迁移波动**：旧页权重转移期约 2–4 周，监控 GSC 收录与展示。
- **island 体积**：工具页性能预算内加载 Vue；超预算则评估静态预渲染方案。
- **匿名滥用**：限流已有（30/min），异常时收紧。
- **内容折旧**：年度专题需持续更新机制，否则降权。
- **生成风险**：公开层不做 LLM 产物（公开工具层边界原则）。

## 7. 数据与复检

- **数据源**：Google Search Console（`sc-domain:liki.hk`）、Bing Webmaster、Google / Bing / 百度 Suggest。
- **工具**：`seo` CLI（report / crawl / page / research），本地 wrapper 已配代理与 Bing key。
- **判定标准**：新页面收录 ≤ 4 周；目标词进入前 20 位后按 CTR 数据迭代 title/description。
- **别名复查时点（WP8 遗留）**：上线 T+4–8 周后，用 GSC 数据确认 `bazi` / `mingshu` 301 与 App 侧旧参数别名（bazi/mingshu/bond）已无流量，再评估移除；在此之前别名保留为兼容层。
