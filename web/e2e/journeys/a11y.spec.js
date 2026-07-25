// Accessibility tests — runs axe-core on key pages and reports violations.
// Does NOT fail the build by default; violations are informational.
// Run with: npx playwright test --config e2e/playwright.config.js a11y.spec.js

import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// Pages to audit with their framework render marker.
const PAGES = [
  { path: '/zh-Hans/naming', marker: 'main',    name: 'naming' },
  { path: '/zh-Hans/naming/chat',     marker: '#view-login',   name: 'chat (login)' },
  { path: '/zh-Hans/mingshu', marker: 'main',  name: 'mingshu' },
  { path: '/zh-Hans/hepan',    marker: 'main',  name: 'hepan' },
  // Legal pages: sr-only heading rendered via compile-time i18n, wait for any visible element
  { path: '/zh-Hans/about',    marker: 'main',    name: 'about' },
  { path: '/zh-Hans/disclaimer', marker: 'main',  name: 'disclaimer' },
  { path: '/zh-Hans/privacy',  marker: 'main',    name: 'privacy' },
  { path: '/zh-Hans/terms',    marker: 'main',    name: 'terms' },
];

test.describe('Accessibility audit', () => {
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
      const byImpact = { critical: [], serious: [], moderate: [], minor: [] };
      const nodeDetails = [];
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
