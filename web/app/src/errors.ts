// 全局致命错误状态：组件崩溃时显示友好错误页，而非白屏。
import { toUserMessage } from '@/utils/errors'
import { reactive } from 'vue'

export const fatal = reactive({
  crashed: false,
  message: '',
})

export function reportFatal(error: unknown) {
  fatal.message = toUserMessage(error, '发生了意外错误。')
  fatal.crashed = true
}