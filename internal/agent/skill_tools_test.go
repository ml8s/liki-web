package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// schema 加载：skill-tools.json 存在且 5 工具全部注册（OpenAI function 格式）。
func TestSkillToolsRegister(t *testing.T) {
	r := NewToolRegistry("") // engine 不参与本测试
	p := NewPythonToolRegistry("../../web/skills")
	if err := p.Register(r); err != nil {
		t.Fatalf("register skill tools: %v", err)
	}
	want := []string{"full_paipan", "liunian", "make_factors", "make_liunian_factors", "query"}
	got := map[string]bool{}
	for _, s := range r.Schemas() {
		var fn struct{ Name string `json:"name"` }
		if err := json.Unmarshal(s.Function, &fn); err == nil {
			got[fn.Name] = true
		}
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("工具未注册: %s (schemas=%v)", w, got)
		}
	}
}

// handler 真跑 python 子进程：make_factors 假 pan 缺 fac → 错误透传（不 panic）。
func TestSkillToolHandler_ErrorPropagation(t *testing.T) {
	r := NewToolRegistry("")
	p := NewPythonToolRegistry("../../web/skills")
	if err := p.Register(r); err != nil {
		t.Fatalf("register: %v", err)
	}
	res, err := r.Execute(context.Background(), "make_factors", json.RawMessage(`{"pan":{}}`))
	if err == nil {
		t.Fatalf("缺 fac 应报错，got %s", res)
	}
	if !strings.Contains(err.Error(), "missing arg: 'fac'") {
		t.Errorf("错误信息未透传 agent_cli: %v", err)
	}
}

// handler 成功路径：query 空快照 → 空结果透传。
func TestSkillToolHandler_QueryEmpty(t *testing.T) {
	r := NewToolRegistry("")
	p := NewPythonToolRegistry("../../web/skills")
	if err := p.Register(r); err != nil {
		t.Fatalf("register: %v", err)
	}
	res, err := r.Execute(context.Background(), "query",
		json.RawMessage(`{"rule":"marriage","snapshots":{"八字":{},"紫微":{}}}`))
	if err != nil {
		t.Fatalf("query 空快照应成功: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatalf("解析结果: %v", err)
	}
	if out["八字"] == nil || out["紫微"] == nil {
		t.Errorf("query 结果缺八字/紫微键: %s", res)
	}
}
