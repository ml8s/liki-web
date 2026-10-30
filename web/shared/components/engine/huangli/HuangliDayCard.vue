<template>
  <details
    :class="['day', variant]"
    @toggle="onToggle($event, day.date)"
  >
    <summary
      class="d-summary"
      @click="emit('select', day.date)"
    >
      <span
        class="d-caret"
        aria-hidden="true"
      />
      <span class="d-date">{{ day.date }}</span>
      <span
        v-if="riZhuText"
        class="d-rizhu font-serif"
      >{{ riZhuText }}</span>
      <span class="d-sub">{{ subText }}</span>
    </summary>
    <HuangliDayDetail :day="day" />
  </details>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 黄历单日卡片：summary 提供日期、日柱与择日原因；展开后复用共享单日详情。
// App 的 HuangliDaysPanel 与营销站择日岛共用。
import { computed } from 'vue'
import type { HuangliDay } from '../../../api/engine/huangli'
import HuangliDayDetail from './HuangliDayDetail.vue'

const props = defineProps<{
  day: HuangliDay
  variant: 'good' | 'possible' | 'bad'
}>()

const emit = defineEmits<{
  select: [date: string]
}>()

function onToggle(event: ToggleEvent, date: string) {
  if ((event.target as HTMLDetailsElement).open) emit('select', date)
}

const riZhuText = computed(() =>
  props.day.ri_zhu ? `${props.day.ri_zhu.gan}${props.day.ri_zhu.zhi}` : '',
)

const subText = computed(() =>
  props.day.reason ? `${props.day.jian_chu} · ${props.day.reason}` : props.day.jian_chu,
)
</script>
