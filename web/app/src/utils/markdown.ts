import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

let linkHooksInstalled = false

function installLinkHooks() {
  if (linkHooksInstalled) return
  linkHooksInstalled = true

  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName !== 'A') return
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer nofollow')
  })
}

export function renderSafeMarkdown(source: string): string {
  installLinkHooks()

  const html = md.render(source ?? '')

  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'p', 'br', 'hr',
      'strong', 'em', 's', 'del',
      'ul', 'ol', 'li',
      'blockquote', 'code', 'pre',
      'table', 'thead', 'tbody', 'tr', 'th', 'td',
      'a', 'span',
    ],
    ALLOWED_ATTR: [
      'href',
      'title',
      'alt',
      'class',
      'start',
      'colspan',
      'rowspan',
    ],
  })
}
