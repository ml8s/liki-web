<script setup lang="ts">
// divination 页免费起卦岛：所问 → 起卦（liuyao_qigua，免登录工具）→ 共享卦象组件。
// 完整解卦（六亲/纳甲/用神）在 App 侧进行——公开工具层只暴露确定性计算。
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { divinationCastingFor, type LiuyaoCasting } from '@shared/domain/divination'
import DivinationCastPanel from '@shared/components/paipan/DivinationCastPanel.vue'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'

const props = defineProps<{
  appUrl: string
  labels: {
    error: string
    cta: string
    privacy: string
  }
}>()

const { t } = useI18n()

const question = ref('')
const loading = ref(false)
const failed = ref(false)
const casting = ref<LiuyaoCasting | null>(null)

async function submit(value: DivinationCastingFormValue) {
  if (loading.value) return
  loading.value = true
  failed.value = false
  casting.value = null
  question.value = value.question.trim()
  try {
    const result = await divinationCastingFor({ mode: value.castingMode })
    casting.value = result.casting
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <DivinationCastingForm
    variant="public-qigua"
    :loading="loading"
    @submit="submit"
  />

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="casting">
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ question ? t('services.divination.questionLabel') + '：' + question : t('services.divination.cast') }}
      </h3>
      <DivinationCastPanel :casting="casting" />
    </div>
    <a class="btn-purple tool-cta" :href="appUrl">{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>

<style scoped>
.tool-form-wide { flex: 1 1 100%; }
</style>
