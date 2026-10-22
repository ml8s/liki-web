// Accessibility gate — axe-core on the marketing pages. Critical and serious
// violations fail the run; moderate/minor ones are printed for triage.
// Run with: npx playwright test --config e2e/playwright.config.ts e2e/journeys/a11y.spec.ts

import { test, expect } from '../fixtures';
import AxeBuilder from '@axe-core/playwright';
import { mockPageview } from '../mocks';

// Pages to audit with their framework render marker.
const PAGES = [
  { path: '/zh-Hans/naming', marker: 'main',    name: 'naming' },
  { path: '/zh-Hans/natal',   marker: 'main',  name: 'natal' },
  { path: '/zh-Hans/hepan',   marker: 'main',  name: 'hepan' },
  { path: '/zh-Hans/liunian', marker: 'main',  name: 'liunian' },
  // Legal pages: sr-only heading rendered via compile-time i18n, wait for any visible element
  { path: '/zh-Hans/about',    marker: 'main',    name: 'about' },
  { path: '/zh-Hans/disclaimer', marker: 'main',  name: 'disclaimer' },
  { path: '/zh-Hans/privacy',  marker: 'main',    name: 'privacy' },
  { path: '/zh-Hans/terms',    marker: 'main',    name: 'terms' },
];

test.describe('Accessibility audit', () => {
  test.beforeEach(async ({ page }) => {
    await mockPageview(page);
  });

  for (const { path, marker, name } of PAGES) {
    test(`${name}: ${path}`, async ({ page }) => {
      await page.goto(path);
      // Wait for FOUC guard to clear (scripts + i18n ready)
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
      await page.waitForSelector(marker, { timeout: 10000 });
      // Ensure no FOUC hiding at the point of audit
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 10000 });

      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .analyze();

      // Group violations by impact for readable output.
      const byImpact: Record<string, string[]> = { critical: [], serious: [], moderate: [], minor: [] };
      const nodeDetails: string[] = [];
      for (const v of results.violations) {
        byImpact[v.impact || 'minor'].push(`${v.id}: ${v.help} (${v.nodes.length} nodes)`);
        if (v.impact === 'critical' || v.impact === 'serious') {
          for (const node of v.nodes) {
            const target = node.target.join(', ');
            const snippet = node.html.slice(0, 120);
            const fgColor = node.all?.[0]?.data?.fgColor || '?';
            const bgColor = node.all?.[0]?.data?.bgColor || '?';
            const contrastRatio = node.all?.[0]?.data?.contrastRatio || '?';
            nodeDetails.push(`  - ${target}
    ratio=${contrastRatio} fg=${fgColor} bg=${bgColor}
    html=${snippet}`);
          }
        }
      }

      const criticalSerious = [...byImpact.critical, ...byImpact.serious];
      if (criticalSerious.length > 0) {
        console.warn(`\n  A11Y [${name}] critical/serious violations:`);
        for (const v of criticalSerious) {
          console.warn(`    - ${v}`);
        }
        console.warn('  Node details:');
        console.warn(nodeDetails.join('\n'));
      }

      // Only assert no critical/serious violations.
      // Moderate/minor are too noisy for CI and tracked separately.
      expect(criticalSerious, `A11Y violations on ${name}`).toEqual([]);
    });
  }
});
