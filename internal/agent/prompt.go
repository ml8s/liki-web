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

// BuildScopedChatPrompt loads one active skill for a chat session. The
// backend still scopes tools; this makes the instruction surface match the
// same entitlement boundary.
func BuildScopedChatPrompt(skillsDir, skillID string) (string, error) {
	if !skillEnabled(skillID) {
		return "", fmt.Errorf("agent: unknown unified chat skill %q", skillID)
	}
	path := filepath.Join(skillsDir, skillID, "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("agent: read scoped chat skill %s: %w", skillID, err)
	}
	prompt := strings.Join([]string{
		"你是 Liki Chat。只服务当前订单已开通且当前会话绑定的 skill。",
		"不要跨入其它 skill 的工具、流程或术语。当前 skill 是 " + skillID + "。",
		"不要读取 webapp 目录；报告生成提示词不属于 Chat。",
		"排盘、因子、断语、卦象、择日和风水盘必须来自工具。意图不明时先澄清。",
		"\n---\n\n# " + skillID + "\n\n" + strings.TrimSpace(string(raw)),
	}, "\n\n")
	if strings.Contains(prompt, "webapp/") {
		return "", fmt.Errorf("agent: scoped chat prompt unexpectedly references webapp")
	}
	return prompt, nil
}

func skillEnabled(skillID string) bool {
	for _, item := range UnifiedChatSkills {
		if item == skillID {
			return true
		}
	}
	return false
}
