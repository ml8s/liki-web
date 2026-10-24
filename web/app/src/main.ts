import './boot'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
// JS 程序化调用的组件（ElMessage/ElMessageBox）样式：显式引入。按需样式经
// 异步 chunk 分割时可能只在部分路由加载，弹窗会退化为裸样式。
import './styles/element-plus.css'
import App from './App.vue'
import router from './router'
import { i18n, initializeAppLocale } from './i18n'
import { reportFatal } from './errors'
import { reportClientError } from '@shared/utils/client-error'
import './styles/app.css'
import './styles/admin.css'

// 异步 rejection（网络超时 / fire-and-forget 失败）→ 记录但**不崩溃**。
// 工业 AA 标准：fatal boundary 只给渲染错误；异步错误降级为日志 + store 内部 error state。
window.addEventListener('unhandledrejection', event => {
  console.error('[Liki] 未处理 Promise 拒绝:', event.reason)
  reportClientError({ code: 'unhandled_rejection' })
  event.preventDefault()
})
window.addEventListener('error', event => {
  reportClientError({ code: 'uncaught_error', detail: event.message })
})

async function bootstrap() {
  await initializeAppLocale()
  const app = createApp(App)
  app.use(createPinia())
  app.use(i18n)
  app.use(router)

  // 渲染/模板错误 → 组件树不可恢复 → fatal 错误页（防白屏）。
  app.config.errorHandler = (error, _instance, info) => {
    console.error('[Liki] 未捕获异常:', error, info)
    reportClientError({ code: 'uncaught_error' })
    reportFatal(error)
  }
  app.mount('#app')
}

void bootstrap().catch(error => {
  console.error('[Liki] 启动失败:', error)
  reportFatal(error)
})
