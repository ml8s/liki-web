<template>
  <div class="mingua-row">
    <div class="mingua font-serif">
      {{ bazhai.ming_gua.gua.name }}
    </div>
    <div>
      <div class="mg-name">
        {{ bazhai.ming_gua.group }}
      </div>
      <div
        v-if="bazhai.ming_gua.gua.wuxing"
        class="mg-wuxing"
      >
        {{ $t('services.fengshui.element') }}{{ bazhai.ming_gua.gua.wuxing }}
      </div>
    </div>
  </div>
  <div class="dirs">
    <div
      v-for="d in dirs"
      :key="d.key"
      :class="['dir', { good: d.good }]"
    >
      <div class="dg font-serif">
        {{ d.gua }}
      </div>
      <div class="dname">
        {{ $t(`profiles.chart.direction.${d.key}`) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BAZHAI_AUS } from '../../utils/bazhai'
import type { BazhaiChart } from '../../api/engine'

const props = defineProps<{ bazhai: BazhaiChart }>()

const dirs = computed(() =>
  Object.entries(props.bazhai.ba_zhai_dirs).map(([key, values]) => ({
    key, gua: (values ?? [''])[0], good: BAZHAI_AUS.includes(key),
  })),
)
</script>

<style scoped>
.mingua-row { display: flex; align-items: center; gap: var(--space-4); margin-bottom: var(--space-4); }
.mingua { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 50%; border: 2px solid var(--liki-primary); font-size: var(--font-size-title-lg); color: var(--liki-primary); background: var(--liki-primary-tint); }
.mg-name { font-size: var(--font-size-body-lg); font-weight: 600; }
.mg-wuxing { font-size: var(--font-size-label); color: var(--liki-text-2); margin-top: 2px; }
.dirs { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-2); }
.dir { border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: var(--space-3); text-align: center; }
.dir .dg { font-size: var(--font-size-title-sm); color: var(--liki-primary); }
.dir.good { border-color: var(--color-success-border); background: var(--color-success-surface); }
.dir.good .dg { color: var(--color-success); }
.dir .dname { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: 3px; }
@media (max-width: 640px) { .dirs { grid-template-columns: repeat(2, 1fr); } }
</style>
