import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { createRecord, deleteRecord, fetchRecords, renameRecord, type RecordKind, type ToolRecord } from '@/api/records'
import { toUserMessage } from '@/utils/errors'
import { confirmDeleteEntity, promptRenameEntity } from '@/utils/dialogs'

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
    let value: string | null
    try {
      value = await promptRenameEntity(t('records.noun'), record.title)
    } catch (err) {
      error.value = toUserMessage(err, t('common.actionFailed'))
      return
    }
    if (value === null) return
    try {
      const updated = await renameRecord(record.id, value)
      const index = records.value.findIndex(item => item.id === record.id)
      if (index >= 0) records.value[index] = updated
    } catch (err) {
      error.value = toUserMessage(err, t('common.actionFailed'))
    }
  }

  async function confirmRemove(record: ToolRecord) {
    if (!await confirmDeleteEntity(t('records.noun'), record.title)) return
    try {
      await deleteRecord(record.id)
      records.value = records.value.filter(item => item.id !== record.id)
    } catch (err) {
      error.value = toUserMessage(err, t('common.actionFailed'))
    }
  }

  return { records, loading, error, load, save, confirmRename, confirmRemove }
}
