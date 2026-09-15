// 全局致命错误状态：组件崩溃时显示友好错误页，而非白屏。
import { reactive } from 'vue'

export const fatal = reactive({
  crashed: false,
  message: '',
})

export function reportFatal(error: unknown) {
  fatal.message = error instanceof Error ? error.message : String(error)
  fatal.crashed = true
}