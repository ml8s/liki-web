package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UnifiedChatSkills are the four installable chat skills. Report prompts under
// webapp/ intentionally never enter this list; they belong to the report
// generator pipeline.
var UnifiedChatSkills = []string{
	"liki-naming",
	"liki-bazi",
	"liki-divination",
	"liki-fengshui",
}

// BuildUnifiedSkillPrompt deterministically loads only the four chat skill
// entry files. It must never scan the skills tree or load webapp report prompts.
func BuildUnifiedSkillPrompt(skillsDir string) (string, error) {
	parts := make([]string, 0, len(UnifiedChatSkills)+1)
	parts = append(parts, `你是 Liki Chat。

你可以服务以下四个 skill：
- liki-naming：起名、改名、外国人起中文名、名字评估。
- liki-bazi：八字、紫微、大运、流年、命书相关问题。
- liki-divination：六爻、奇门、黄历、具体问事。
- liki-fengshui：阳宅风水、八宅、玄空。

规则：
- 根据用户当前订单允许的 skill 工作。
- 只使用当前订单允许领域的方法与术语。
- 不要读取 webapp 目录；报告生成提示词不属于 Chat。
- 排盘、因子、断语、卦象、择日和风水盘必须来自工具。
- 意图不明时先澄清。
`)

	for _, skillID := range UnifiedChatSkills {
		if filepath.Base(skillID) != skillID || strings.ContainsAny(skillID, "/\\") {
			return "", fmt.Errorf("agent: invalid skill id %q", skillID)
		}
		path := filepath.Join(skillsDir, skillID, "SKILL.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("agent: read unified skill %s: %w", skillID, err)
		}
		parts = append(parts, "---\n\n# "+skillID+"\n\n"+strings.TrimSpace(string(raw)))
	}

	prompt := strings.Join(parts, "\n\n")
	if strings.Contains(prompt, "webapp/") {
		return "", fmt.Errorf("agent: unified chat prompt unexpectedly references webapp")
	}
	return prompt, nil
}
