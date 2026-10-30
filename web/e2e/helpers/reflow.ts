import { expect, type Page } from '@playwright/test'

export interface ReflowContractOptions {
  viewportWidth: number
  scopeSelector: string
  controlSelectors: string[]
  /** Scopes such as the domain strip may intentionally scroll within themselves. */
  scrollableScope?: boolean
}

/**
 * WCAG reflow is not just document scrollWidth: a shell with `overflow-x:
 * hidden` can hide clipping. Therefore each primary control must also remain
 * inside the current viewport.
 */
export async function expectReflowContract(
  page: Page,
  { viewportWidth, scopeSelector, controlSelectors, scrollableScope = false }: ReflowContractOptions,
) {
  const scope = page.locator(scopeSelector)
  await expect(scope).toBeVisible()
  const violations: string[] = []
  await expect
    .poll(async () => {
      violations.length = 0
      const scopeOverflow = await scope.evaluate(node => node.scrollWidth - node.clientWidth)
      if (!scrollableScope && scopeOverflow > 1) {
        violations.push(`${scopeSelector} overflows by ${scopeOverflow}px`)
      }
      const documentOverflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      if (documentOverflow > 1) violations.push(`document overflows by ${documentOverflow}px`)

      for (const selector of controlSelectors) {
        const control = page.locator(selector).first()
        await expect(control).toBeVisible()
        const box = await control.boundingBox()
        if (!box) {
          violations.push(`${selector} has no layout box`)
          continue
        }
        if (box.x < -1) violations.push(`${selector} starts offscreen at ${box.x}px`)
        if (box.x + box.width > viewportWidth + 1) {
          violations.push(`${selector} ends at ${box.x + box.width}px beyond ${viewportWidth}px`)
        }
      }
      if (violations.length) console.error('[REFLOW]', violations.join(' | '))
      return violations.length
    })
    .toBe(0)

  expect(violations, violations.join('\n')).toEqual([])
}
