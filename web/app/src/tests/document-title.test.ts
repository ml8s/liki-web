import { describe, expect, it } from 'vitest'
import { formatDocumentTitle, setDocumentTitle } from '@/utils/documentTitle'
import { setAppLocale } from '@/i18n'

describe('document title', () => {
  it('joins the page name and the site name', () => {
    expect(formatDocumentTitle('工作台', 'Liki')).toBe('工作台 · Liki')
  })

  it('omits the suffix when the page name already carries the site name', () => {
    expect(formatDocumentTitle('Liki Chat', 'Liki')).toBe('Liki Chat')
  })

  it('shows only the site name when there is no page name', () => {
    expect(formatDocumentTitle(undefined, 'Liki')).toBe('Liki')
  })

  it('writes the composed title to the document', async () => {
    await setAppLocale('zh-Hant', { persist: false })
    setDocumentTitle('工作台')
    expect(document.title).toBe('工作台 · Liki')
  })
})
