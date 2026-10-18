import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { createRecord, deleteRecord, fetchRecords, renameRecord, type RecordKind, type ToolRecord } from '@/api/records'
import { toUserMessage } from '@/utils/errors'

/**
 * 工具记录标准回路（问卦/择日/合盘/八宅共用）：
 * 加载列表、保存、重命名（确认框）、删除（确认框），成功后本地更新列表。
 */
export function useToolRecords(kind: RecordKind) {
  const { t } = useI18n()
  const records = ref<ToolRecord[]>([])
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      records.value = await fetchRecords(kind)
    } catch (err) {
      error.value = toUserMessage(err, t('records.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  async function save(title: string, input: unknown, output?: unknown) {
    try {
      const record = await createRecord(kind, title, input, output)
      records.value = [record, ...records.value]
    } catch (err) {
      error.value = toUserMessage(err, t('records.saveFailed'))
    }
  }

  async function confirmRename(record: ToolRecord) {
    let value: string
    try {
      const result = await ElMessageBox.prompt(
        t('records.renamePrompt'),
        t('records.renameTitle'),
        {
          inputValue: record.title,
          inputPlaceholder: t('records.renamePlaceholder'),
          inputValidator: (raw: string) => {
            const title = raw.trim()
            if (!title) return t('records.renameRequired')
            if (title.length > 60) return t('records.renameTooLong')
            return true
          },
          confirmButtonText: t('common.save'),
          cancelButtonText: t('common.cancel'),
        },
      )
      value = result.value.trim()
    } catch (err) {
      if (err === 'cancel' || err === 'close') return
      error.value = toUserMessage(err, t('records.actionFailed'))
      return
    }
    try {
      const updated = await renameRecord(record.id, value)
      const index = records.value.findIndex(item => item.id === record.id)
      if (index >= 0) records.value[index] = updated
    } catch (err) {
      error.value = toUserMessage(err, t('records.actionFailed'))
    }
  }

  async function confirmRemove(record: ToolRecord) {
    try {
      await ElMessageBox.confirm(
        t('records.deleteConfirm', { title: record.title }),
        t('records.deleteTitle'),
        { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') },
      )
    } catch {
      return
    }
    try {
      await deleteRecord(record.id)
      records.value = records.value.filter(item => item.id !== record.id)
    } catch (err) {
      error.value = toUserMessage(err, t('records.actionFailed'))
    }
  }

  return { records, loading, error, load, save, confirmRename, confirmRemove }
}
