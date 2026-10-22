// 营销工具岛的通用挂载：SSR 直出的 data-props 容器由 Vue 就地 hydration。
// 独立外部模块（CSP 禁 inline script）；共享组件所需的 profiles.chart.* /
// services.* 文案按 locale 拆文件动态加载（构建期分 chunk，首访只拉当前语言；
// 见 scripts/build-chart-messages.js 与 check-chart-messages.js）。
import { createSSRApp, type Component } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { ID_INJECTION_KEY } from 'element-plus'

interface IslandProps {
  locale: 'zh-Hans' | 'zh-Hant' | 'en'
  [key: string]: unknown
}

// 三种语言的文案 chunk 由 Vite 按模板动态 import 静态枚举（构建产物各一份）。
function loadChartMessages(locale: IslandProps['locale']) {
  switch (locale) {
    case 'zh-Hans':
      return import('@shared/i18n/chart-messages.zh-Hans.json')
    case 'zh-Hant':
      return import('@shared/i18n/chart-messages.zh-Hant.json')
    default:
      return import('@shared/i18n/chart-messages.en.json')
  }
}

export function mountIsland(id: string, component: Component) {
  const el = document.getElementById(id)
  const raw = el?.dataset.props
  if (!el || !raw) return
  const props = JSON.parse(raw) as IslandProps
  void (async () => {
    const chartMessages = (await loadChartMessages(props.locale)).default
    const i18n = createI18n({
      legacy: false,
      locale: props.locale,
      fallbackLocale: 'en',
      messages: { [props.locale]: chartMessages },
      globalInjection: true,
      missingWarn: false,
      fallbackWarn: false,
    })
  const app = createSSRApp(component, props)
  // Astro prerenders islands; Element Plus requires a deterministic SSR id seed.
  app.provide(ID_INJECTION_KEY, { prefix: 1024, current: 0 })
  app.use(createPinia()).use(i18n).mount(el)
  })()
}
