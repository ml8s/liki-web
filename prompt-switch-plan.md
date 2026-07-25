# naming chat 切换 prompt 到 naming/SKILL.md 实施计划

## 影响范围分析

### 不受影响 ✅
- **mingshu 报告生成**：`Generator.process()` 使用 `generate.md` 作为 system prompt，完全不依赖 agent.prompt
- **hepan 报告生成**：同上
- **API 路径**：不变
- **其他产品页面**：不变
- **npx skills install**：naming/SKILL.md 已有 `name: liki-naming`，可作为子 skill 安装

### 受影响 ⚡
- **naming chat**：system prompt 从 SKILL.md 改为 naming/SKILL.md
- **read_file 路径引用**：naming/SKILL.md 中 `../bazi/` 需要改为 `bazi/`

---

## 实施步骤

### Step 1: 修复 naming/SKILL.md 中的路径

当前文件中有 `../bazi/knowledge/yongshen.md`：
```
参考 `../bazi/knowledge/yongshen.md`
```

read_file 的 root 是 `/web/skills`，`filepath.IsLocal` 拒绝 `..` 路径。
改为：
```
参考 `bazi/knowledge/yongshen.md`
```

**不影响作为子 skill：** 子 skill 安装后文件在 `{skills_dir}/liki-naming/SKILL.md`，但子 skill 中引用路径是指向安装目录的相对路径。然而 naming/SKILL.md 作为 chat 的 prompt 时，read_file root 固定为 `/web/skills`，所以路径必须相对于 `/web/skills`。

### Step 2: 修改 main.go

```diff
- PromptFile: envOr("LIKI_PROMPT_FILE", "SKILL.md"),
+ PromptFile: envOr("LIKI_PROMPT_FILE", "naming/SKILL.md"),
```

或新增一个专门的 naming prompt：

```go
// 原有 base prompt 用于报告生成（实际上报告生成不用它）
// naming chat 用 naming/SKILL.md
namingPromptFile := envOr("LIKI_NAMIING_PROMPT_FILE", "naming/SKILL.md")
namingPrompt, err := os.ReadFile(filepath.Join(cfg.SkillsDir, namingPromptFile))
```

但选项 1 更简单：直接改默认值。

### Step 3: 更新 .env.example

```
- LIKI_PROMPT_FILE=SKILL.md
+ LIKI_PROMPT_FILE=naming/SKILL.md
```

### Step 4: 部署检查

Dockerfile 中 `COPY web/ /web/` 包含 `web/skills/naming/SKILL.md`，已就位。

---

## 验证清单

- [ ] `go build ./...` 编译通过
- [ ] 启动服务，naming chat 使用 naming/SKILL.md
- [ ] naming chat 中 LLM 能调用 `read_file("bazi/knowledge/yongshen.md")`
- [ ] mingshu 报告生成正常（不受影响）
- [ ] hepan 报告生成正常（不受影响）
- [ ] `npx skills add liki.hk naming` 可安装子 skill
