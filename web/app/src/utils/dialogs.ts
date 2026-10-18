// 记录/会话等资源共用的对话框回路：重命名 prompt 与破坏性操作 confirm。
// 文案统一走 common.*（{target} 为已翻译的资源名词），调用方只传数据。
import { ElMessageBox } from 'element-plus'
import { i18n } from '@/i18n'

function t(key: string, values?: Record<string, unknown>): string {
  return i18n.global.t(key, values ?? {})
}

/** 重命名对话框：返回 trim 后的标题；取消/关闭返回 null。 */
export async function promptRenameEntity(noun: string, current: string): Promise<string | null> {
  try {
    const result = await ElMessageBox.prompt(
      t('common.renamePrompt', { target: noun }),
      t('common.renameTitle', { target: noun }),
      {
        inputValue: current,
        inputPlaceholder: t('common.renamePlaceholder', { target: noun }),
        inputValidator: (raw: string) => {
          const value = raw.trim()
          if (!value) return t('common.renameRequired')
          if (value.length > 60) return t('common.renameTooLong')
          return true
        },
        confirmButtonText: t('common.save'),
        cancelButtonText: t('common.cancel'),
      },
    )
    return result.value.trim()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return null
    throw error
  }
}

/** 删除确认对话框：确认返回 true，取消返回 false。 */
export async function confirmDeleteEntity(noun: string, title: string): Promise<boolean> {
  return confirmAction({
    title: t('common.deleteTitle', { target: noun }),
    message: t('common.deleteConfirm', { title }),
    confirmText: t('common.delete'),
  })
}

export interface ConfirmActionOptions {
  title: string
  message: string
  confirmText: string
}

/** 破坏性操作通用确认（warning 风格，取消文案统一 common.cancel）。 */
export async function confirmAction(options: ConfirmActionOptions): Promise<boolean> {
  try {
    await ElMessageBox.confirm(options.message, options.title, {
      type: 'warning',
      confirmButtonText: options.confirmText,
      cancelButtonText: t('common.cancel'),
    })
    return true
  } catch {
    return false
  }
}
