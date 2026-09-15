import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/theme-chalk/base.css'
import App from './App.vue'
import router from './router'
import { reportFatal } from './errors'
import './styles/app.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// 未捕获异常/异步拒绝 → 错误边界（防白屏）
app.config.errorHandler = (error, _instance, info) => {
  console.error('[Liki] 未捕获异常:', error, info)
  reportFatal(error)
}
window.addEventListener('unhandledrejection', event => {
  console.error('[Liki] 未处理 Promise 拒绝:', event.reason)
  reportFatal(event.reason)
})

app.mount('#app')