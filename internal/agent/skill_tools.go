// skill_tools.go — liki skill 规则引擎工具（python 子进程适配器）。
//
// skill 的 5 个规则引擎工具（full_paipan/liunian/make_factors/make_liunian_factors/query）
// 由 python 工具链实现（tools/*.py，纯标准库）。web agent 无 shell，Go 通过
// exec python3 tools/agent_cli.py 调用（stdin {fn,args} → stdout {ok,data|error}）。
// 工具 schema（OpenAI function calling 格式）从 skill 根 tools/skill-tools.json 加载——
// 单一来源，外部 agent 与本地 agent 共用同一份定义。
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// skillToolsJSON 是 5 个规则引擎工具的 schema 文件（相对 SkillsDir）。
const skillToolsJSON = "liki-bazi/tools/skill-tools.json"

// agentCliPy 是 python 工具链的 CLI 分派入口（相对 SkillsDir）。
const agentCliPy = "liki-bazi/tools/agent_cli.py"

// PythonTimeout 单次 python 子进程调用上限。
const PythonTimeout = 30 * time.Second

// PythonToolRegistry 注册并执行 skill 规则引擎工具。
type PythonToolRegistry struct {
	SkillsDir string // skills 根目录（如 web/skills）
	PythonBin string // python 解释器（默认 python3）
}

// NewPythonToolRegistry 构造规则引擎工具注册器。
func NewPythonToolRegistry(skillsDir string) *PythonToolRegistry {
	return &PythonToolRegistry{SkillsDir: skillsDir, PythonBin: "python3"}
}

// Register 从 skill-tools.json 加载 5 工具 schema 并注册到 ChatToolRegistry。
// schema 文件缺失/非法时返回错误（调用方决定是否继续——工具不注册则规则引擎不可用）。
func (p *PythonToolRegistry) Register(r *ChatToolRegistry) error {
	schemaPath := filepath.Join(p.SkillsDir, skillToolsJSON)
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read skill tools schema: %w", err)
	}
	var doc struct {
		Tools []struct {
			Type     string          `json:"type"`
			Function json.RawMessage `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse skill tools schema: %w", err)
	}
	registered := 0
	for _, t := range doc.Tools {
		if t.Type != "function" {
			continue
		}
		var fn struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		}
		if err := json.Unmarshal(t.Function, &fn); err != nil || fn.Name == "" {
			continue
		}
		name := fn.Name
		r.Register(name, fn.Description, fn.Parameters, p.handler(name))
		registered++
	}
	slog.Info("registered skill tools", "count", registered, "schema", schemaPath)
	return nil
}

// handler 构造单个工具的执行器：exec python3 agent_cli.py，stdin 传 {fn,args}。
// 参数化调用（无代码拼接）；超时由 ctx 控制；python 缺失/失败返回 error（不降级）。
func (p *PythonToolRegistry) handler(fn string) func(context.Context, json.RawMessage) (json.RawMessage, error) {
	cliPath := filepath.Join(p.SkillsDir, agentCliPy)
	return func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
		req := map[string]any{"fn": fn, "args": args}
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("%s: marshal args: %w", fn, err)
		}

		callCtx, cancel := context.WithTimeout(ctx, PythonTimeout)
		defer cancel()
		cmd := exec.CommandContext(callCtx, p.PythonBin, cliPath)
		cmd.Stdin = bytes.NewReader(reqJSON)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		// stdout 上限（防异常大输出耗尽内存——正常断语 JSON 远小于此）
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("%s: stdout pipe: %w", fn, err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("%s: python start: %w (stderr: %s)", fn, err, stderr.String())
		}
		const maxStdout = 10 << 20 // 10MB
		raw, readErr := io.ReadAll(io.LimitReader(stdoutPipe, maxStdout+1))
		if waitErr := cmd.Wait(); waitErr != nil && readErr == nil {
			readErr = waitErr
		}
		if readErr != nil {
			return nil, fmt.Errorf("%s: python exec: %w (stderr: %s)", fn, readErr, stderr.String())
		}
		if len(raw) > maxStdout {
			return nil, fmt.Errorf("%s: python 输出超上限 %dMB（疑似异常，拒绝处理）", fn, maxStdout>>20)
		}
		stdout := bytes.NewReader(raw)

		var resp struct {
			OK    bool            `json:"ok"`
			Data  json.RawMessage `json:"data"`
			Error string          `json:"error"`
		}
		if err := json.NewDecoder(stdout).Decode(&resp); err != nil {
			return nil, fmt.Errorf("%s: parse agent_cli output: %w", fn, err)
		}
		if !resp.OK {
			return nil, fmt.Errorf("%s: %s", fn, resp.Error)
		}
		return resp.Data, nil
	}
}
