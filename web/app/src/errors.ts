// 全局致命错误状态：组件崩溃时显示友好错误页，而非白屏。
import { toUserMessage } from '@/utils/errors'
import { reactive } from 'vue'
import { i18n } from '@/i18n'

export const fatal = reactive({
  crashed: false,
  message: '',
})

export function reportFatal(error: unknown) {
  fatal.message = toUserMessage(error, i18n.global.t('fatal.defaultMessage'))
  fatal.crashed = true
}
