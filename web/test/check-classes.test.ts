import { describe, expect, it } from 'vitest';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = fileURLToPath(new URL('../..', import.meta.url));
const SCRIPT = path.join(ROOT, 'web/scripts/check-classes.js');
const INVALID_TEMPLATE = path.join(ROOT, 'web/src/_temp_invalid.astro');

// 检查脚本用非零退出码报失败，消息在 stderr；这里统一成可直接断言的结果。
function runCheck(): { status: number; stderr: string } {
  try {
    execFileSync(process.execPath, [SCRIPT], { cwd: ROOT, stdio: ['ignore', 'pipe', 'pipe'] });
    return { status: 0, stderr: '' };
  } catch (error) {
    const { status, stderr } = error as { status?: number | null; stderr?: string | Buffer };
    return { status: status ?? 1, stderr: String(stderr ?? '') };
  }
}

describe('check-classes.js', () => {
  it('passes when every template class resolves in app.css', () => {
    expect(runCheck().status).toBe(0);
  });

  it('fails and names the offending class when a template uses an undefined class', () => {
    fs.writeFileSync(INVALID_TEMPLATE, '<div class="this-class-does-not-exist"></div>');
    try {
      const { status, stderr } = runCheck();
      expect(status).toBe(1);
      expect(stderr).toContain('this-class-does-not-exist');
    } finally {
      fs.unlinkSync(INVALID_TEMPLATE);
    }
  });

  // 契约：模板化属性里的字面量也参与检查——`class={`...`}` 只算 `${...}` 之外的片段
  // （曾经整条跳过，页脚 border-t/py-8 因此长期没定义也没人发现）。
  it('fails on an undefined class inside a templated class attribute', () => {
    fs.writeFileSync(INVALID_TEMPLATE, '<footer class={`mt-6 this-class-does-not-exist${x ? " y" : ""}`}></footer>');
    try {
      const { status, stderr } = runCheck();
      expect(status).toBe(1);
      expect(stderr).toContain('this-class-does-not-exist');
    } finally {
      fs.unlinkSync(INVALID_TEMPLATE);
    }
  });

  it('fails on an undefined class in a ternary branch too', () => {
    fs.writeFileSync(INVALID_TEMPLATE, "<main class={kind === 'article' ? 'mt-6 this-class-does-not-exist' : 'mt-6'}></main>");
    try {
      const { status, stderr } = runCheck();
      expect(status).toBe(1);
      expect(stderr).toContain('this-class-does-not-exist');
    } finally {
      fs.unlinkSync(INVALID_TEMPLATE);
    }
  });

  // 比较运算里的字符串是数据（kind === 'article'），不是类名：不能误报。
  it('ignores comparison operands in a templated class attribute', () => {
    fs.writeFileSync(INVALID_TEMPLATE, "<main class={kind === 'article' ? 'mt-6' : 'mt-6'}></main>");
    try {
      expect(runCheck().status).toBe(0);
    } finally {
      fs.unlinkSync(INVALID_TEMPLATE);
    }
  });
});
