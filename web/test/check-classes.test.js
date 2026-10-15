import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';
import path from 'path';

const ROOT = path.resolve(__dirname, '..', '..');
const SCRIPT = path.join(ROOT, 'web/scripts/check-classes.js');

describe('check-classes.js', () => {
  it('exits 0 when all templates use valid CSS classes', () => {
    const out = execSync(`node "${SCRIPT}"`, { cwd: ROOT, encoding: 'utf-8', stdio: ['pipe', 'pipe', 'ignore'] });
    expect(out).toContain('✓');
  });

  it('fails when encountering an undefined class', () => {
    // Create a temp template with an invalid class, run check, verify failure
    const tmpDir = ROOT + '/web/src';
    const orig = tmpDir + '/_temp_invalid.astro';
    require('fs').writeFileSync(orig, '<div class="this-class-does-not-exist"></div>');
    try {
      const out = execSync(`node "${SCRIPT}"`, { cwd: ROOT, encoding: 'utf-8', stdio: ['pipe', 'pipe', 'ignore'] });
      expect(out).toContain('❌');
    } catch (e) {
      expect(e.status).toBe(1);
    } finally {
      require('fs').unlinkSync(orig);
    }
  });
});
