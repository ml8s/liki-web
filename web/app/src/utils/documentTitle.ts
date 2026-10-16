// 浏览器标签标题：静态页与 SPA 共用同一格式「<页面名> · <站点名>」。
// 页面名由调用方给出（路由 meta.title/titleKey）；没有页面名时只显示站点名。
import { i18n } from '@/i18n'

export function formatDocumentTitle(page: string | undefined, site: string): string {
  if (!page) return site
  // 页面名里已经带了站点名（例如 “Liki Chat”、首页 slogan）就不再重复追加。
  return page.includes(site) ? page : `${page} · ${site}`
}

export function setDocumentTitle(page?: string): void {
  document.title = formatDocumentTitle(page, i18n.global.t('site.name'))
}
