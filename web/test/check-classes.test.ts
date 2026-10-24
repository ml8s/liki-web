import { describe, expect, it } from 'vitest';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import os from 'node:os';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const WEB_ROOT = fileURLToPath(new URL('../', import.meta.url));
const SCRIPT = path.join(REPO_ROOT, 'web/scripts/check-classes.js');

// 检查脚本用非零退出码报失败，消息在 stderr；这里统一成可直接断言的结果。
function runCheck(projectRoot = WEB_ROOT): { status: number; stdout: string; stderr: string } {
  try {
    const result = spawnSync(process.execPath, [SCRIPT, projectRoot], { cwd: REPO_ROOT, encoding: 'utf8' });
    if (result.error) return { status: result.status ?? 1, stdout: result.stdout ?? '', stderr: String(result.error) };
    return { status: result.status ?? 1, stdout: result.stdout ?? '', stderr: result.stderr ?? '' };
  } catch (error) {
    return { status: 1, stdout: '', stderr: String(error) };
  }
}

describe('check-classes.js', () => {
  it('passes when every template class resolves in app.css', () => {
    expect(runCheck().status).toBe(0);
  });

  function isolatedProject(): string {
    const project = fs.mkdtempSync(path.join(os.tmpdir(), 'check-classes-'));
    fs.mkdirSync(path.join(project, 'css'), { recursive: true });
    fs.mkdirSync(path.join(project, 'src'), { recursive: true });
    fs.copyFileSync(path.join(WEB_ROOT, 'css/app.css'), path.join(project, 'css/app.css'));
    return project;
  }

  it('fails and names the offending class when a template uses an undefined class', () => {
    const project = isolatedProject();
    const invalidTemplate = path.join(project, 'src/invalid.astro');
    fs.writeFileSync(invalidTemplate, '<div class="this-class-does-not-exist"></div>');
    const { status, stderr } = runCheck(project);
    expect(status).toBe(1);
    expect(stderr).toContain('this-class-does-not-exist');
  });

  // 契约：模板化属性里的字面量也参与检查——`class={`...`}` 只算 `${...}` 之外的片段。
  it('fails on an undefined class inside a templated class attribute', () => {
    const project = isolatedProject();
    const invalidTemplate = path.join(project, 'src/invalid.astro');
    fs.writeFileSync(invalidTemplate, '<footer class={`mt-6 this-class-does-not-exist${x ? " y" : ""}`}></footer>');
    const { status, stderr } = runCheck(project);
    expect(status).toBe(1);
    expect(stderr).toContain('this-class-does-not-exist');
  });

  it('fails on an undefined class in a ternary branch too', () => {
    const project = isolatedProject();
    const invalidTemplate = path.join(project, 'src/invalid.astro');
    fs.writeFileSync(invalidTemplate, "<main class={kind === 'article' ? 'mt-6 this-class-does-not-exist' : 'mt-6'}></main>");
    const { status, stderr } = runCheck(project);
    expect(status).toBe(1);
    expect(stderr).toContain('this-class-does-not-exist');
  });

  // 比较运算里的字符串是数据（kind === 'article'），不是类名：不能误报。
  it('ignores comparison operands in a templated class attribute', () => {
    expect(runCheck().status).toBe(0);
  });
});
