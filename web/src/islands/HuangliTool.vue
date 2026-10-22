<script setup lang="ts">
// huangli 页免费择日岛：事项 + 日期范围 → 共享引擎 → 宜/平/忌分组展示。
// 结果组件来自共享层（与 App 的 HuangliView 同源）。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { huangliDaysFor, type HuangliDays } from '@shared/domain/huangli'
import { EVENT_KEYS } from '@shared/utils/huangli'
import HuangliDaysPanel from '@shared/components/charts/HuangliDaysPanel.vue'
import HuangliRangeFields from '@shared/components/huangli/HuangliRangeFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: {
    submit: string
    computing: string
    error: string
    cta: string
    privacy: string
  }
}>()

const { t } = useI18n()

const form = reactive({ event: 'wedding', startDate: '', endDate: '' })
const loading = ref(false)
const failed = ref(false)
const result = ref<HuangliDays | null>(null)

const spanDays = computed(() => {
  if (!form.startDate || !form.endDate) return 0
  return Math.round((new Date(form.endDate).getTime() - new Date(form.startDate).getTime()) / 86400000) + 1
})

async function submit() {
  if (!form.startDate || !form.endDate || loading.value) return
  const count = spanDays.value
  if (form.endDate < form.startDate || count <= 0 || count > 30) {
    failed.value = true
    return
  }
  loading.value = true
  failed.value = false
  result.value = null
  try {
    result.value = await huangliDaysFor(form.event, form.startDate, count)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <FormShell
    form-class="tool-form"
    :model="form"
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <HuangliRangeFields :value="form" @update:patch="Object.assign(form, $event)" />
  </FormShell>

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="result">
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.huangli.resultTitle', { event: t(`services.huangli.events.${form.event}`) }) }}
      </h3>
      <HuangliDaysPanel :days="result" />
    </div>
    <a class="btn-purple tool-cta" :href="appUrl">{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
