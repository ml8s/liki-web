package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UnifiedChatSkills is the single installable chat skill. The engine's unified
// skill routes 八字/起名/问卦/风水 internally; report prompts under webapp/
// intentionally never enter this list.
var UnifiedChatSkills = []string{
	"liki",
}

// BuildScopedChatPrompt loads the unified chat skill for a chat session. The
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
		"你是 Liki Chat，服务统一命理 skill。按用户意图路由到八字/紫微、起名、问卦、风水等领域。",
		"领域不明确时，先用一个问题确认主目标；不要跨入未开通或无关的能力。",
		"不要读取 webapp 目录；报告生成提示词不属于 Chat。",
		"排盘、因子、断语、卦象、择日和风水盘必须来自工具。",
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
