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
	"strings"
	"time"
)

// PythonSkills are the domain tool packages inside the unified skill that ship
// a Python tool layer (bazi, divination). naming/fengshui use direct RPC only.
var PythonSkills = []string{"liki/bazi", "liki/divination"}

const PythonTimeout = 30 * time.Second

type PythonToolRegistry struct {
	SkillsDir string
	PythonBin string
}

func NewPythonToolRegistry(skillsDir string) *PythonToolRegistry {
	return &PythonToolRegistry{SkillsDir: skillsDir, PythonBin: "python3"}
}

type manifest struct {
	Tools []struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	} `json:"tools"`
}

func (p *PythonToolRegistry) Register(r *ChatToolRegistry) error {
	return p.RegisterSkills(r, PythonSkills...)
}

func (p *PythonToolRegistry) RegisterSkills(r *ChatToolRegistry, skillIDs ...string) error {
	seen := make(map[string]bool)
	count := 0

	for _, skillID := range skillIDs {
		if skillID == "" || filepath.IsAbs(skillID) || strings.Contains(skillID, "..") {
			return fmt.Errorf("agent: invalid python skill id %q", skillID)
		}
		manifestPath := filepath.Join(p.SkillsDir, skillID, "tools", "skill-tools.json")
		cliPath := filepath.Join(p.SkillsDir, skillID, "tools", "agent_cli.py")
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("read skill tools schema: %w", err)
		}

		var manifestData manifest
		if err := json.Unmarshal(raw, &manifestData); err != nil {
			return fmt.Errorf("parse skill tools schema: %w", err)
		}

		for _, tool := range manifestData.Tools {
			if tool.Type != "function" || tool.Function.Name == "" {
				continue
			}
			if seen[tool.Function.Name] {
				continue
			}
			seen[tool.Function.Name] = true
			r.Register(tool.Function.Name, tool.Function.Description, tool.Function.Parameters, p.executor(cliPath, tool.Function.Name))
			count++
		}
	}

	slog.Info("registered python skill tools", "count", count)
	return nil
}

func (p *PythonToolRegistry) executor(cliPath, fn string) func(context.Context, json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
		request := struct {
			Fn   string          `json:"fn"`
			Args json.RawMessage `json:"args"`
		}{Fn: fn, Args: args}
		requestJSON, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("%s: marshal args: %w", fn, err)
		}

		callCtx, cancel := context.WithTimeout(ctx, PythonTimeout)
		defer cancel()
		cmd := exec.CommandContext(callCtx, p.PythonBin, cliPath)
		cmd.Stdin = bytes.NewReader(requestJSON)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("%s: stdout pipe: %w", fn, err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("%s: start: %w (stderr: %s)", fn, err, stderr.String())
		}

		const maxStdout = 10 << 20
		raw, readErr := io.ReadAll(io.LimitReader(stdout, maxStdout+1))
		waitErr := cmd.Wait()
		if readErr == nil {
			readErr = waitErr
		}
		if readErr != nil {
			return nil, fmt.Errorf("%s: python exec: %w (stderr: %s)", fn, readErr, stderr.String())
		}
		if len(raw) > maxStdout {
			return nil, fmt.Errorf("%s: stdout exceeds %d MB", fn, maxStdout>>20)
		}

		var response struct {
			OK    bool            `json:"ok"`
			Data  json.RawMessage `json:"data"`
			Error string          `json:"error"`
		}
		if err := json.NewDecoder(strings.NewReader(string(raw))).Decode(&response); err != nil {
			return nil, fmt.Errorf("%s: parse CLI output: %w", fn, err)
		}
		if !response.OK {
			return nil, fmt.Errorf("%s: %s", fn, response.Error)
		}
		return response.Data, nil
	}
}
