import { expect, test } from '../fixtures'
import { mockPageview } from '../mocks'

test.describe('shared input contract', () => {
  test.use({ viewport: { width: 390, height: 1000 } })

  test.beforeEach(async ({ page }) => {
    await mockPageview(page)
  })

  test('Element Plus fields draw one box and one focus ring', async ({ page }) => {
    await page.goto('/zh-Hant/')
    const tool = page.locator('#quick-birth-tool')
    await expect(tool.locator('.el-form')).toBeVisible()

    const dateInput = tool.locator('.el-date-editor--date input')
    const dateWrapper = tool.locator('.el-date-editor--date .el-input__wrapper')
    await expect(dateInput).toBeVisible()

    const idle = await dateWrapper.evaluate(el => {
      const style = getComputedStyle(el)
      const inner = el.querySelector<HTMLInputElement>('.el-input__inner')
      const innerStyle = inner ? getComputedStyle(inner) : null
      return {
        height: el.getBoundingClientRect().height,
        shadow: style.boxShadow,
        innerBorder: innerStyle?.borderTopWidth,
        innerShadow: innerStyle?.boxShadow,
      }
    })
    expect(idle.height).toBeGreaterThanOrEqual(44)
    expect(idle.innerBorder).toBe('0px')
    expect(idle.innerShadow).toBe('none')
    expect(idle.shadow).toContain('0px 0px 0px 1px')

    await dateInput.focus()
    await expect
      .poll(() => dateWrapper.evaluate(el => getComputedStyle(el).boxShadow))
      .toContain('0px 0px 0px 3px')

    const placeholder = await dateInput.evaluate(el => {
      const style = getComputedStyle(el, '::placeholder')
      const surface = el.closest<HTMLElement>('.el-input__wrapper') ?? el
      const background = getComputedStyle(surface).backgroundColor
      const parse = (value: string) => {
        const match = value.match(/rgba?\(([^)]+)\)/)
        if (!match) return { r: 255, g: 255, b: 255 }
        const [r, g, b] = match[1].split(/[,/\s]+/).filter(Boolean).map(Number)
        return { r, g, b }
      }
      const luminance = ({ r, g, b }: { r: number; g: number; b: number }) => {
        const channel = (value: number) => {
          const normalized = value / 255
          return normalized <= 0.03928
            ? normalized / 12.92
            : ((normalized + 0.055) / 1.055) ** 2.4
        }
        return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
      }
      const fg = parse(style.color)
      const bg = parse(background)
      const ratio = (Math.max(luminance(fg), luminance(bg)) + 0.05)
        / (Math.min(luminance(fg), luminance(bg)) + 0.05)
      return ratio
    })
    expect(placeholder).toBeGreaterThanOrEqual(4.5)
  })

  test('choice cards meet touch and selected-state contracts', async ({ page }) => {
    await page.goto('/zh-Hant/')
    const tool = page.locator('#quick-birth-tool')
    const cards = tool.locator('.choice-card .el-radio-button__inner')
    await expect(cards).toHaveCount(2)

    const cardHeight = await cards.first().evaluate(el => el.getBoundingClientRect().height)
    expect(cardHeight).toBeGreaterThanOrEqual(44)

    await cards.nth(1).click()
    await expect(tool.locator('.choice-card .el-radio-button.is-active')).toHaveCount(1)
    await expect(tool.locator('.choice-card .choice-card__state--selected')).toHaveCount(1)
    await expect
      .poll(() => cards.nth(1).evaluate(el => getComputedStyle(el).backgroundColor))
      .not.toBe('rgb(255, 255, 255)')
    await expect
      .poll(() => cards.nth(1).evaluate(el => getComputedStyle(el).boxShadow))
      .toContain('0px 0px 0px 1px')

    const states = await cards.evaluateAll(elements => elements.map(el => {
      const style = getComputedStyle(el)
      return {
        background: style.backgroundColor,
        color: style.color,
        shadow: style.boxShadow,
        text: el.textContent?.trim(),
      }
    }))
    expect(states[1]?.background).not.toBe(states[0]?.background)
  })

  test('place autocomplete offers ambiguous candidates without a second select', async ({ page }) => {
    await page.goto('/zh-Hant/')
    const tool = page.locator('#quick-birth-tool')
    const place = tool.getByRole('textbox', { name: /出生地/ })
    await place.fill('长沙')

    const options = page.locator('.lf-region-autocomplete li[role="option"]')
    await expect(options.first()).toBeVisible()
    await expect(options).toHaveCount(2)
    await expect(tool.locator('.region-status')).toContainText(/多個同名地點|同名地点|Multiple places/)

    await options.first().click()
    await expect(place).toHaveValue(/长沙市/)

    await tool.locator('.region-manual-toggle').click()
    await expect(tool.locator('input[name="birth-longitude"]')).toHaveValue(/112\.98/)
    await expect(tool.locator('input[name="birth-timezone"]')).toHaveValue('Asia/Shanghai')
  })
})
