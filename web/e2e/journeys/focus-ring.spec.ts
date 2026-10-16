// 焦点视觉只能有一层。
//
// 静态页的 input/select 在鼠标点选时同样命中 :focus-visible（浏览器对文本类控件的
// 规则），App 里 Element Plus 还会给 .el-input__inner 加 tabindex="0" 命中全局
// [tabindex] 分支。只要全局规则不排除原生控件，控件自己的 ring 里就会再套一圈
// outline——输入框看上去是「两个框」。反过来，自绘控件（div[tabindex]）没有组件
// ring，全局 outline 是键盘用户唯一的焦点线索，不能被收走。
import type { Page } from '@playwright/test'
import { test, expect } from '../fixtures'
import { mockPageview, mockWorkspace } from '../mocks'

test.use({ locale: 'zh-Hant' })

// 走真实键盘路径：:focus-visible 只在键盘交互下成立，JS focus() 测不出键盘用户的焦点态。
async function tabTo(page: Page, selector: string, max = 30) {
  for (let i = 0; i < max; i++) {
    await page.keyboard.press('Tab')
    const hit = await page.evaluate(sel => document.activeElement?.matches(sel) ?? false, selector)
    if (hit) return
  }
  throw new Error(`Tab 没有走到 ${selector}`)
}

test('static shell: one focus ring per control', async ({ page }) => {
  await mockPageview(page)
  await page.goto('/zh-Hant/')

  // 按钮没有自己的 ring，键盘焦点必须由全局 outline 提供（含 class 名单会漏掉的提交按钮）。
  await tabTo(page, '.paipan-hook button[type="submit"]')
  const submit = page.locator('.paipan-hook button[type="submit"]')
  await expect(submit).toHaveCSS('outline-style', 'solid')
  await expect(submit).toHaveCSS('box-shadow', 'none')

  // 表单控件：ring 由自己的 box-shadow 画，不能再叠一层 outline。
  const input = page.locator('.paipan-hook input[name="date"]')
  await input.click()
  await expect(input).toHaveCSS('outline-style', 'none')
  await expect(input).toHaveCSS('box-shadow', /3px/)

  const select = page.locator('.paipan-hook select')
  await select.click()
  await expect(select).toHaveCSS('outline-style', 'none')
  await expect(select).toHaveCSS('box-shadow', /3px/)
})

test('app shell: native controls rely on Element Plus, custom controls keep the outline', async ({ page }) => {
  await mockWorkspace(page, {})
  await page.goto('/app/birth-profiles/new')
  await page.waitForSelector('.birth-form')

  // 返回按钮是自绘的 div[tabindex="0"]，没有组件 ring，键盘 outline 不能被收走。
  await tabTo(page, '.el-page-header__back')
  const back = page.locator('.el-page-header__back')
  await expect(back).toHaveCSS('outline-style', 'solid')
  await expect(back).toHaveCSS('box-shadow', 'none')

  // EL 输入框：wrapper 是 ring 的持有者，内部原生 input 是 tabindex="0"，不能再挂 outline。
  await page.locator('.birth-form .el-input__inner').first().click()
  await expect(page.locator('.birth-form .el-input__wrapper').first()).toHaveCSS('box-shadow', /inset/)
  await expect(page.locator('.birth-form .el-input__inner').first()).toHaveCSS('outline-style', 'none')
})
