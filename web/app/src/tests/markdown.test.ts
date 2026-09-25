import { describe, expect, it } from 'vitest'
import { renderSafeMarkdown } from '@/utils/markdown'

describe('renderSafeMarkdown', () => {
  it('renders basic markdown', () => {
    const html = renderSafeMarkdown('# Title\n\n**important**')
    expect(html).toContain('<h1>Title</h1>')
    expect(html).toContain('<strong>important</strong>')
  })

  it('removes dangerous html and javascript urls', () => {
    const html = renderSafeMarkdown('<script>alert(1)</script>\n\n[click](javascript:alert(1))')
    expect(html).not.toContain('<script')
    expect(html).not.toContain('href="javascript:')
  })

  it('hardens external links', () => {
    const html = renderSafeMarkdown('[Liki](https://liki.hk)')
    expect(html).toContain('rel="noopener noreferrer nofollow"')
    expect(html).toContain('target="_blank"')
  })
})
