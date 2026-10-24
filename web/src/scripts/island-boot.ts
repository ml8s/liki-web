// 营销工具岛的通用挂载：SSR 直出的 data-props 容器由 Vue 就地 hydration。
// 独立外部模块（CSP 禁 inline script）；共享组件所需的 profiles.chart.* /
// services.* 文案按 locale 拆文件动态加载（构建期分 chunk，首访只拉当前语言；
// 见 scripts/build-chart-messages.js 与 check-chart-messages.js）。
import { createSSRApp, h, type Component } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { ElConfigProvider, ID_INJECTION_KEY } from 'element-plus'
import en from 'element-plus/es/locale/lang/en'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import { reportClientError } from '@shared/utils/client-error'

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

function installClientErrorReporting() {
  const state = window as unknown as { __likiErrorReporting?: boolean }
  if (state.__likiErrorReporting) return
  state.__likiErrorReporting = true
  window.addEventListener('unhandledrejection', () => {
    reportClientError({ code: 'unhandled_rejection' })
  })
  window.addEventListener('error', event => {
    reportClientError({ code: 'uncaught_error', detail: event.message })
  })
}

export function mountIsland(id: string, component: Component) {
  installClientErrorReporting()
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
  const elementLocale = props.locale === 'zh-Hans' ? zhCn : props.locale === 'en' ? en : zhTw
  // Element Plus date/time pickers need the same locale as the page. The
  // product App already provides this at its root; marketing islands need it
  // too, otherwise zh-Hant pages render English month/weekday names.
  const Root = {
    render: () => h(ElConfigProvider, { locale: elementLocale }, () => h(component, props)),
  }
  const app = createSSRApp(Root)
  // Astro prerenders islands; Element Plus requires a deterministic SSR id seed.
  app.provide(ID_INJECTION_KEY, { prefix: 1024, current: 0 })
  app.use(createPinia()).use(i18n).mount(el)
  })()
}
