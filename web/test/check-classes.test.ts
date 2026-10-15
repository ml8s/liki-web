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
});
