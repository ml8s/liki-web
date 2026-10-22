<template>
  <div class="liuyao-panel">
    <div class="gua-row">
      <div class="gua-name font-serif">
        {{ reading.benGua }}
      </div>
      <div
        v-if="reading.bianGua"
        class="gua-sub"
      >
        {{ $t('services.divination.changed') }} {{ reading.bianGua }}
      </div>
    </div>
    <div class="yao-stack">
      <div
        v-for="(line, idx) in lines"
        :key="idx"
        class="yao"
      >
        <span class="yao-idx">{{ line.position }}</span>
        <span :class="['yao-shape', line.yang ? 'yang' : 'yin']" />
        <span
          v-if="line.dong"
          class="yao-dong"
        >{{ $t('services.divination.moving') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 六爻起卦结果展示（免费层）：本卦/变卦名 + 六爻爻线（自下而上，动爻标注）。
// 由 casting 推演（见 shared/domain/divination.ts）；完整解卦（六亲/纳甲）在 App 侧。
import { computed } from 'vue'
import { readCasting, type YiGuaReading } from '../../domain/divination'
import type { LiuyaoCasting } from '../../api/engine/liuyao'

const props = defineProps<{ casting: LiuyaoCasting }>()

const reading = computed<YiGuaReading>(() => readCasting(props.casting))

const lines = computed(() =>
  reading.value.yaos
    .map((value, i) => ({
      position: i + 1,
      yang: value === 7 || value === 9,
      dong: reading.value.dong[i],
    }))
    .reverse(),
)
</script>

<style scoped>
.gua-row { display: flex; align-items: baseline; gap: var(--space-3); margin-bottom: var(--space-3); }
.gua-name { font-size: var(--font-size-title-md); font-weight: 700; color: var(--liki-primary); }
.gua-sub { font-size: var(--font-size-label); color: var(--liki-text-2); }
.yao-stack { display: flex; flex-direction: column; gap: var(--space-2); max-width: 320px; }
.yao { display: flex; align-items: center; gap: var(--space-3); }
.yao-idx { flex: 0 0 1.4em; font-size: var(--font-size-caption); color: var(--liki-text-3); font-variant-numeric: tabular-nums; }
.yao-shape { flex: 1; height: 8px; border-radius: var(--radius-small); background: var(--liki-ink); }
.yao-shape.yin {
  background: linear-gradient(
    to right,
    var(--liki-ink) 0 42%,
    transparent 42% 58%,
    var(--liki-ink) 58% 100%
  );
}
.yao-dong { flex: 0 0 auto; font-size: var(--font-size-micro); color: var(--liki-primary); }
.yao:has(.yao-dong) .yao-shape { box-shadow: 0 0 0 2px var(--liki-primary-alpha-10); }
</style>
