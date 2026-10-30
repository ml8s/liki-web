# 择日产品页与流日增强计划

Status: backend flow-day contracts are aligned and the public `/huangli/`
tool now offers optional selected-day Bazi and Zi Wei flow-day evidence. The
remaining roadmap is the full product-page/content polish, consolidated report,
and any independent Date Selection agent decision.

## 1. Summary

将现有 `/[locale]/huangli/` 从“免费黄历查询能力页”升级为正式的“择日产品页”，并规划后续个人化择日能力。

产品分层如下：

1. **通用择日层**
   - 基于事件、日期范围和老黄历规则筛选候选日期。
   - 无需登录，不需要出生信息。
   - 输出宜忌、建除、黄黑道、冲煞、日柱、纳音等确定性证据。

2. **选中日港证据层**
   - 用户选中某个候选日后，展开该日更完整的证据。
   - 继续以确定性规则为主，不伪造结论。

3. **个人化流日层**
   - 结合临时出生事实查看该日对当事人的流日关系。
   - 使用 engine 的 `bazi_liuri` 和 `ziwei_liuri`；出生信息不持久化。

4. **App 深入解读层**
   - 结合事件目标、个人命盘、大运流年、家庭成员和时间约束，给出可执行择日建议。

## 2. SEO 定位

### Canonical URL

继续使用现有 URL，不新增重复路径：

```txt
/zh-Hans/huangli/
/zh-Hant/huangli/
/en/huangli/
```

原因：

- `/huangli/` 已有既有入口和搜索语义。
- 新建 `/zeri/` 或 `/date-selection/` 会稀释择日主题权重。
- 产品页应升级现有页面，而不是另建重复工具页。

### 关键词方向

#### zh-Hans

主关键词：

- 择日
- 老黄历择日
- 黄道吉日
- 结婚吉日
- 搬家吉日
- 开业吉日

辅助关键词：

- 嫁娶择日
- 入宅择日
- 签约择日
- 动土择日
- 老黄历宜忌
- 建除十二神

#### zh-Hant

主关键词：

- 擇日
- 老黃曆擇日
- 黃道吉日
- 結婚吉日
- 搬家吉日
- 開業吉日

#### en

主关键词：

- Chinese date selection
- auspicious dates
- Chinese almanac
- wedding date selection
- moving date selection
- business opening date

辅助关键词：

- Yi/Ji
- Day Officer
- clash zodiac
- Yellow Calendar
- personal date selection

### 标题草案

#### zh-Hans

```txt
专业择日｜老黄历吉日、流日与个人化日期选择 · Liki
```

#### zh-Hant

```txt
專業擇日｜老黃曆吉日、流日與個人化日期選擇 · Liki
```

#### en

```txt
Chinese Date Selection: Almanac, Flow-Day Timing & Personal Auspicious Dates | Liki
```

### H1 草案

#### zh-Hans

```txt
专业择日：老黄历吉日、流日证据与个人化日期选择
```

#### zh-Hant

```txt
專業擇日：老黃曆吉日、流日證據與個人化日期選擇
```

#### en

```txt
Professional Date Selection: Almanac Days, Flow-Day Evidence, and Personal Timing
```

## 3. 页面信息架构

### 目标顺序

```txt
Hero
→ 免费择日工具
→ 候选日期结果
→ 选中日证据
→ 择日方法
→ 个人化流日路线
→ 常见场景
→ FAQ
→ App CTA
```

### 3.1 Hero

目标：让用户立刻理解这是“按事件择日”，不是只查“今天宜什么”。

内容：

```txt
H1：专业择日：老黄历吉日、流日证据与个人化日期选择
说明：按婚礼、开业、搬家、签约、动土等真实事件筛选候选日期，并逐步查看黄历与流日证据。
主 CTA：开始择日
次 CTA：了解择日方法
```

### 3.2 免费择日工具

保留并强化现有工具：

输入：

- 事件类型
- 开始日期
- 结束日期

输出：

- recommended / possible / unsuitable 分组
- 每日干支
- 日柱
- 纳音
- 建除十二神
- 黄道/黑道
- 宜忌
- 冲煞
- 吉神方位
- suitability 原因

交互要求：

- 默认事件可以保留 wedding，但结果区要强调“按事件筛选”。
- 日期范围校验完整。
- 结果分组可展开。
- 移动端日期卡片不横向溢出。
- 工具提交后才请求引擎。

### 3.3 候选日期结果

结果区改为三层：

```text
概览
→ 推荐日
→ 可行日
→ 不宜日
```

每个日期至少展示：

- 日期
- 干支
- 日柱
- 建除
- 黄黑道
- 与事件匹配的 suitability
- 主要 reason

展开后展示：

- 纳音
- 冲煞
- 吉神方位
- 详细宜忌
- 后续流日证据入口

### 3.4 选中日证据

用户点击某个日期后，展开该日完整证据。

第一阶段展示现有黄历证据：

```txt
当日干支
日柱与纳音
建除十二神
黄道/黑道
宜忌
冲煞生肖
吉神方位
事件匹配原因
```

第二阶段规划个人化流日：

```txt
当日与当事人命局的关系
当日干支与日主十神
与年柱、月柱、日柱、时柱的互动
与大运、流年的关系
可用时间窗口
需避开的冲煞
```

第三阶段规划紫微流日：

```txt
流日命宫
宫位主题
四化落点
重要星曜提示
```

紫微流日已通过 `ziwei_liuri` 契约上线；缺失数据保持显式 partial/error 状态。

## 4. 择日专业模型

择日不应只被描述为“看老黄历宜忌”。产品页和 App 都要建立分层模型。

### 4.1 事件约束层

先确认真实事件：

- 婚嫁
- 开业
- 搬家 / 入宅
- 签约
- 动土 / 修造
- 出行
- 安葬
- 其他用户自定义事件

每个事件的约束不同。例如：

| 事件 | 择日重点 |
| --- | --- |
| 婚嫁 | 双方命局、冲煞、家庭时间约束 |
| 开业 | 行业五行、财方、开市宜忌、日主稳定度 |
| 搬家 / 入宅 | 坐向、家庭成员、入宅宜忌 |
| 签约 | 日辰稳定度、冲突、谈判窗口 |
| 动土 / 修造 | 坐向、煞方、季节与日课 |

### 4.2 通用黄历层

基于老黄历规则筛选：

- 宜忌
- 建除十二神
- 黄道 / 黑道
- 冲煞
- 吉神方位
- 日柱与纳音

这一层是公开免费工具的核心。

### 4.3 八字流日层

选中日期后，结合当事人出生信息计算流日。

目标输入：

```ts
interface BaziLiuriInput {
  birth: BirthInput
  selectedDate: string
  timezone: string
  longitude?: number
  event?: string
}
```

目标输出应包含：

```ts
interface BaziLiuriOutput {
  selectedDate: string
  dayPillar: Pillar
  tenGods: Array<{
    target: string
    relation: string
  }>
  natalInteractions: Array<{
    pillar: 'year' | 'month' | 'day' | 'hour'
    relation: string
    detail?: string
  }>
  dayunInteractions: Array<{
    period: string
    relation: string
    detail?: string
  }>
  liunianInteractions: Array<{
    year: number
    relation: string
    detail?: string
  }>
  shensha: Array<{
    name: string
    category?: string
    description?: string
  }>
  cautions: string[]
}
```

具体 DTO 以后端契约为准，前端不能自行拼造命理结论。

### 4.4 紫微流日层

如果引擎支持，再增加紫微流日：

目标输入：

```ts
interface ZiweiLiuriInput {
  birth: BirthInput
  selectedDate: string
  timezone: string
  longitude?: number
  event?: string
}
```

目标输出可能包含：

```ts
interface ZiweiLiuriOutput {
  selectedDate: string
  dayPalace: string
  palaces: Array<{
    name: string
    stars: string[]
    theme?: string
  }>
  siHua: Record<string, string>
  siHuaPalaces?: Record<string, string>
  cautions?: string[]
}
```

该层必须等待 `ziwei_liuri` 后端契约明确后再实施。

### 4.5 综合择日结果

最终综合结果不应该是简单分数，而应呈现证据结构：

```txt
黄历层：
该日期满足 / 不满足哪些事件规则。

八字流日层：
该日干支与当事人命局、大运、流年的关系。

紫微流日层：
该日宫位主题、四化与重要星曜提示。

注意层：
需要避开的冲煞、重复出现的忌项、不确定项。
```

如果未来要引入“综合评分”，必须有明确算法和引擎契约，不能前端随机加权。

## 5. 页面内容架构

### 5.1 Hero

```txt
H1：专业择日：老黄历吉日、流日证据与个人化日期选择
说明：按婚礼、开业、搬家、签约、动土等真实事件筛选候选日期，并逐步查看黄历与流日证据。
主 CTA：开始择日
次 CTA：了解择日方法
```

### 5.2 Free Tool

放在页面上方。

```txt
标题：免费择日工具
说明：选择事件和日期范围，先筛选老黄历候选日。
输入：事件、开始日期、结束日期
输出：recommended / possible / unsuitable
```

### 5.3 Selected Day Evidence

```txt
标题：选中日证据
说明：点击候选日期，查看当日干支、建除、黄黑道、冲煞与事件匹配原因。
状态：
- 第一阶段：通用黄历证据
- 第二阶段：登录用户或临时输入出生信息后的八字流日
- 第三阶段：紫微流日
```

### 5.4 Methodology

```txt
标题：择日不只是看黄历宜忌
内容：
1. 先确认事件与真实约束。
2. 用老黄历筛选候选日期。
3. 查看每日干支、建除、黄黑道与冲煞。
4. 结合当事人命局查看流日关系。
5. 必要时进入 App 结合大运、流年与事件目标深入解读。
```

### 5.5 Event Use Cases

分场景内容：

#### 婚嫁择日

```txt
重点：双方命局、冲煞、家庭时间约束。
展示：黄历宜忌 + 流日与双方命局关系。
```

#### 开业择日

```txt
重点：行业五行、财方、开市宜忌、日主稳定度。
展示：黄历规则 + 当日与决策者命局关系。
```

#### 搬家 / 入宅

```txt
重点：坐向、家庭成员、入宅宜忌。
展示：黄历规则 + 择日窗口。
```

#### 签约 / 重要决策

```txt
重点：日辰稳定度、冲突、谈判窗口。
展示：黄历规则 + 当日十神与流日关系。
```

#### 动土 / 修造

```txt
重点：坐向、煞方、季节与日课。
展示：黄历规则 + 后续个人化流日扩展。
```

### 5.6 FAQ

至少覆盖：

1. 择日和看老黄历有什么区别？
2. 老黄历写“宜嫁娶”，所有人都可以用吗？
3. 择日为什么还要看流日？
4. 婚嫁择日需要几个人的出生信息？
5. 开业择日为什么关注行业和决策者？
6. 搬家和入宅择日一样吗？
7. 免费择日结果会保存吗？
8. App 择日解读有什么不同？
9. 你们会如何保护出生信息？

FAQ 需要输出 `FAQPage` structured data。

## 6. SEO 方案

### 6.1 Canonical

继续使用：

```txt
/zh-Hans/huangli/
/zh-Hant/huangli/
/en/huangli/
```

不新增：

```txt
/zeri/
/date-selection/
/jiri/
```

避免重复内容和权重分散。

### 6.2 hreflang

```html
<link rel="alternate" hreflang="zh-Hans" href="https://liki.hk/zh-Hans/huangli/" />
<link rel="alternate" hreflang="zh-Hant" href="https://liki.hk/zh-Hant/huangli/" />
<link rel="alternate" hreflang="en" href="https://liki.hk/en/huangli/" />
```

`x-default` 按现有站点策略确认。

### 6.3 Metadata

#### zh-Hans title

```txt
专业择日｜老黄历吉日、流日与个人化日期选择 · Liki
```

#### zh-Hant title

```txt
專業擇日｜老黃曆吉日、流日與個人化日期選擇 · Liki
```

#### en title

```txt
Chinese Date Selection: Almanac, Flow-Day Timing & Personal Auspicious Dates | Liki
```

#### zh-Hans description

```txt
按婚礼、开业、搬家、签约、动土等事件选择吉日。查看老黄历宜忌、建除十二神、冲煞与候选日期比较，并规划结合八字流日与个人命盘的深入择日。
```

#### en description

```txt
Choose auspicious dates for weddings, business openings, moving, signing, and groundbreaking. Compare Chinese almanac rules, Day Officers, clash, and plan personal flow-day timing in the app.
```

### 6.4 Structured data

应包含：

1. `Service`
2. `BreadcrumbList`
3. `FAQPage`
4. 可选 `WebApplication`

不要添加与页面内容不符的 `Product` 或评分信息。

### 6.5 内链

指向 `/huangli/` 的来源：

1. 首页服务区。
2. natal 页。
3. liunian 页。
4. hepan 页。
5. divination 页。
6. fengshui 页。
7. naming 页。
8. 未来择日指南文章。

择日页指向：

1. `/natal/`：择日前建立个人命盘。
2. `/liunian/`：查看流年节奏。
3. `/app/`：进入个人化择日解读。
4. `/disclaimer/`：免责声明。
5. `/privacy/`：隐私说明。

## 7. 技术实施阶段

### Phase 0：契约与关键词冻结

1. 确认现有 `huangli_days` DTO 是否已满足候选日展示。
2. 梳理选中日详情需要的新字段。
3. 确认 `bazi_liuri` 是否已有后端能力。
4. 确认 `ziwei_liuri` 是否已有后端能力。
5. 确认择日是否采用独立 Date Selection agent。
6. 冻结 SEO keyword map 和页面 IA。

停止条件：

- 如果后端无法提供流日契约，前端只做黄历产品页和路线图，不伪造流日结果。
- 如果无法确认年份/日期边界，不做“综合吉凶评分”。

### Phase 1：择日产品页重构

范围：

1. 升级 `/huangli/` 页面结构。
2. 工具区上移。
3. 增加方法论、证据说明、事件场景和 FAQ。
4. 保持现有免费工具可用。
5. 三语言内容同步。

验收：

- 三语言页面无 raw i18n key。
- H1 唯一。
- canonical / hreflang 正确。
- structured data 可解析。
- 移动端工具可用。
- smoke 无 console error。

### Phase 2：选中日详情

范围：

1. 点击候选日期后展开选中日详情。
2. 展示现有黄历字段。
3. 增加“可复制摘要”或“继续在 App 中解读”入口。
4. URL 不写入出生事实。
5. 保持工具状态在刷新后的合理恢复策略。

验收：

- 每个候选日可展开。
- 推荐日 / 可行日 / 不宜日均能查看详情。
- 关闭详情后焦点合理返回。
- App handoff 不回退。

### Phase 3：八字流日层

Status: implemented on the public selected-day path.

范围：

1. Added the web API adapter and `BaziLiuRiPanel`.
2. Supports temporary birth facts for the selected day.
3. Loading / error states are owned by the shared engine query boundary.
4. Birth facts are client-side and not persisted.
5. Copy is synchronized across zh-Hans, zh-Hant, and English.

验收：

- 展示流日干支、十神、命局互动、大运流年互动和神煞。
- 不在页面里手写命理规则。
- 出生信息不落库，除非用户明确创建档案。

### Phase 4：紫微流日层

Status: implemented on the public selected-day path.

范围：

1. Added the web API adapter and `ZiweiLiuRiPanel`.
2. Supports selected-day Zi Wei palace and Si Hua display.
3. Added artifact catalog, engine contract, and E2E coverage.

验收：

- 展示流日命宫、宫位主题、四化与重要星曜。
- 缺失数据时明确显示 partial，不伪造。
- 面板懒加载，不影响首屏。

### Phase 5：综合择日报告

依赖：

- 黄历、八字流日、紫微流日契约完整。

范围：

1. 将多层证据组织为选中日报告。
2. 分组展示：
   - 黄历规则
   - 八字流日
   - 紫微流日
   - 注意事项
3. 支持进入 App 深入解读。
4. 可选复制摘要。

验收：

- 不展示无依据的综合分数。
- 每条结论都能追溯到确定性字段。
- AI 解读只在 App 中进行。

## 8. App / 后端架构建议

### 当前状态

现有择日能力主要是：

```txt
huangli_days
```

当前产品可以完成通用黄历择日。

### 推荐目标模型

采用此前 section 7 讨论中的 Option B：将择日逐步发展为独立 Date Selection agent。

原因：

1. 择日输入是“事件 + 日期范围 + 当事人约束”，不适合硬塞进 natal。
2. 择日输出是“候选日期 + 选中日证据 + 个人流日关系”，也不是普通对话上下文。
3. 独立 agent 更容易扩展婚礼、开业、搬家、动土等场景。
4. 后续可以增加事件规则、日课约束和个人流日组合，而不会让 natal expert 变成杂项。

### 目标能力

```txt
date_selection.candidates
```

输入：

```ts
{
  event: string
  startDate: string
  endDate: string
  people?: Array<BirthInput>
  constraints?: Record<string, unknown>
}
```

输出：

```ts
{
  items: HuangliDay[]
}
```

### 目标流日能力

```txt
date_selection.flow_day
```

或拆成：

```txt
bazi_liuri
ziwei_liuri
```

具体拆分以后端契约设计为准。

### 隐私要求

1. 未登录临时计算不得持久化出生信息。
2. 登录用户使用已有档案时，遵循现有档案权限。
3. 择日报告如需保存，必须明确保存范围和删除方式。
4. 报告链接不得默认公开。
5. 需要限流，防止公开流日接口被滥用。

## 9. 测试计划

### 9.1 SEO 测试

1. 三语言 title / description 唯一。
2. canonical 正确。
3. hreflang 正确。
4. sitemap 包含择日页。
5. structured data 无错误。
6. OG / Twitter 图片可访问。
7. H1 唯一。
8. H2 / H3 层级清晰。

### 9.2 功能测试

1. 选择事件和日期范围。
2. recommended / possible / unsuitable 分组正确。
3. 选中日详情展开正常。
4. 工具错误态可重试。
5. 移动端无横向溢出。
6. App handoff 正常。
7. 未登录临时计算不持久化出生信息。

### 9.3 回归测试

1. `pages.spec.ts`
2. `huangli-tool.spec.ts`
3. `tool-handoff.spec.ts`
4. `chrome-consistency.spec.ts`
5. smoke 测试
6. sitemap / hreflang 检查

## 10. 验收标准

### 内容验收

1. 页面明确定位为择日产品页。
2. 工具在首屏或近首屏。
3. 解释“择日不只是看黄历宜忌”。
4. 说明黄历层、八字流日层、紫微流日层的差异。
5. 有婚嫁、开业、搬家、签约、动土等场景。
6. 有 App 深入解读路径。
7. FAQ 覆盖核心疑虑。

### SEO 验收

1. `/huangli/` 是择日 canonical 页面。
2. 三语言 metadata 唯一。
3. hreflang 正确。
4. structured data 可解析。
5. sitemap 包含页面。
6. 内链锚文本多样化。

### 工程验收

1. lint / typecheck / build 通过。
2. E2E 通过。
3. 移动端无横向溢出。
4. 工具结果不伪造。
5. 未登录出生信息不持久化。
6. 新增面板懒加载。

## 11. 交付里程碑

| 里程碑 | 内容 | 依赖 |
| --- | --- | --- |
| M0 | 契约、关键词、IA 冻结 | 后端确认 + SEO 关键词确认 |
| M1 | 择日产品页重构 | 现有 huangli_days |
| M2 | 选中日详情 | 现有 DTO 或小扩展 |
| M3 | 八字流日 | `bazi_liuri` 后端契约 |
| M4 | 紫微流日 | `ziwei_liuri` 后端契约 |
| M5 | 综合择日报告 | 前四项完成 |
| M6 | SEO / E2E / 文档验收 | 全部功能完成 |

## 12. 风险与停止条件

### 风险

1. 后端没有 `bazi_liuri` / `ziwei_liuri` 能力。
2. 择日 agent 引入第五个领域，会影响现有“四域 IA”断言。
3. 流日计算涉及出生信息，隐私边界更敏感。
4. 综合择日容易被误读为吉凶评分。
5. 三语言专业术语需要统一。

### 停止条件

出现以下情况时先回到设计，不继续实施：

1. 后端无法给出稳定的流日 DTO。
2. 无法明确日期/时区/年界规则。
3. 无法保证未登录出生信息不持久化。
4. 前端需要伪造流日关系或综合评分。
5. 择日 agent 的路由和 IA 变更没有评审结论。

## 13. 文档更新

本计划确认后应同步更新：

1. `docs/frontend/marketing-free-tools-audit-2026-10-30.md`
   - 将 `/huangli/` 从“已 parity”更新为“产品页升级 + 流日路线已规划”。
2. `docs/frontend/section7-decisions.md`
   - Item 8 / Item 9 增加 2026-10-30 planning addendum。
3. `FRONTEND.md`
   - 择日页从 capability page 升级为 product page。
4. `docs/frontend/ux-architecture.md`
   - 增加择日产品页 IA 与 canonical 策略。
