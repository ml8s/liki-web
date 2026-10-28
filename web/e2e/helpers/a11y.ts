import AxeBuilder from '@axe-core/playwright'
import { expect, type Page } from '@playwright/test'

/** Contract baseline: WCAG 2.x A/AA critical and serious violations fail. */
export async function expectNoCriticalA11yViolations(page: Page, label: string) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze()
  const violations = results.violations.filter(
    violation => violation.impact === 'critical' || violation.impact === 'serious',
  )

  const detail = violations
    .map(violation => `${violation.id}: ${violation.nodes.length} nodes [${violation.nodes
      .map(node => node.target.join(' '))
      .join('; ')}]`)
    .join('\n')

  expect(violations, `a11y violations on ${label}:\n${detail}`).toEqual([])
}
