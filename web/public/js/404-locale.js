(() => {
  const path = window.location.pathname
  const locale = /^\/en(\/|$)/.test(path)
    ? 'en'
    : /^\/zh-Hans(\/|$)/.test(path)
      ? 'zh-Hans'
      : 'zh-Hant'
  const copy = {
    'zh-Hant': {
      lang: 'zh-Hant',
      title: '頁面未找到 | liki.hk',
      heading: '頁面未找到',
      slogan: '懂命理，用 Liki',
      hint: '您訪問的頁面不存在或已被移除。',
      cta: '返回首頁',
    },
    'zh-Hans': {
      lang: 'zh-Hans',
      title: '页面未找到 | liki.hk',
      heading: '页面未找到',
      slogan: '懂命理，用 Liki',
      hint: '您访问的页面不存在或已被移除。',
      cta: '返回首页',
    },
    en: {
      lang: 'en',
      title: 'Page not found | liki.hk',
      heading: 'Page not found',
      slogan: 'For Astrology, use Liki.',
      hint: 'The page you requested does not exist or has been moved.',
      cta: 'Back to home',
    },
  }[locale]

  document.documentElement.lang = copy.lang
  document.title = copy.title
  for (const [key, value] of Object.entries(copy)) {
    if (key === 'lang' || key === 'title') continue
    const el = document.querySelector(`[data-404-${key}]`)
    if (el) el.textContent = value
  }
})()
