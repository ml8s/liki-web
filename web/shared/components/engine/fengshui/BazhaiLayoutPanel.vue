<template>
  <div
    class="bazhai-layout-panel"
    :data-density="density ?? 'full'"
  >
    <div class="bz-verdict">
      <div class="bz-gua-line">
        <span class="bz-gua-name font-serif">{{ result.ming_gua }}</span>
        <span class="bz-group">{{ result.group }}</span>
      </div>
      <div class="mini-table-wrap">
        <table
          class="mini-table bz-table"
          :aria-label="$t('services.fengshui.layoutTitle')"
        >
          <thead>
            <tr>
              <th scope="col" />
              <th scope="col">
                {{ $t('services.fengshui.directionLabel') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.guaLabel') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.youxing') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.rating') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.match') }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in verdictRows"
              :key="row.key"
              :class="{ 'is-focused': row.key === focused }"
              tabindex="0"
              :data-focus-key="row.key"
              :aria-current="row.key === focused ? 'true' : undefined"
            >
              <th scope="row">
                {{ row.label }}
              </th>
              <td>{{ row.info.direction }}</td>
              <td>{{ row.info.gua_name }}</td>
              <td>{{ row.info.youxing }}</td>
              <td :class="row.good ? 'bz-good' : 'bz-bad'">
                {{ row.rating }}
              </td>
              <td :class="row.good ? 'bz-good' : 'bz-bad'">
                {{ row.match }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="bz-note">
        {{ $t('services.fengshui.layoutNote') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BazhaiLayoutPanelProps } from '../contract'

const props = defineProps<BazhaiLayoutPanelProps>()

const { t } = useI18n()

type BazhaiDoorStove = BazhaiLayoutPanelProps['result']['door']

const RATING_KEYS: Record<string, string> = {
  大吉: 'services.fengshui.ratingValue.veryAuspicious',
  吉: 'services.fengshui.ratingValue.auspicious',
  平: 'services.fengshui.ratingValue.neutral',
  凶: 'services.fengshui.ratingValue.inauspicious',
  大凶: 'services.fengshui.ratingValue.veryInauspicious',
}

function labelOf(keys: Record<string, string>, value: string): string {
  return keys[value] ? t(keys[value]) : value
}

const verdictRows = computed(() => {
  const rows: Array<{ key: string; label: string; info: BazhaiDoorStove }> = [
    { key: 'door', label: t('services.fengshui.door'), info: props.result.door },
    { key: 'master', label: t('services.fengshui.master'), info: props.result.master },
    { key: 'stove', label: t('services.fengshui.stove'), info: props.result.stove },
  ]
  return rows.map(row => ({
    ...row,
    rating: labelOf(RATING_KEYS, row.info.rating),
    match: row.info.match === '吉'
      ? t('services.fengshui.ratingValue.auspicious')
      : row.info.match === '凶'
        ? t('services.fengshui.ratingValue.inauspicious')
        : row.info.match,
    good: row.info.match === '吉',
  }))
})
</script>

<style scoped>
.bazhai-layout-panel {
  min-width: 0;
  overflow-wrap: anywhere;
}
.bz-verdict { min-width: 0; margin-top: var(--space-3); }
.bz-gua-line { display: flex; flex-wrap: wrap; align-items: baseline; gap: var(--space-2); margin-bottom: var(--space-2); }
.bz-gua-name { font-size: var(--font-size-title-sm); font-weight: 600; color: var(--liki-ink); }
.bz-group { font-size: var(--font-size-label); color: var(--liki-text-2); }
.mini-table-wrap { min-width: 0; overflow-x: auto; }
.bz-table { table-layout: fixed; }
.bz-table th,
.bz-table td { overflow-wrap: anywhere; white-space: normal; }
.bz-table tbody tr:focus-visible { outline: 2px solid var(--liki-primary); outline-offset: -2px; }
.bz-table tbody tr.is-focused th,
.bz-table tbody tr.is-focused td { background: var(--liki-primary-tint); }
.bz-good { color: var(--color-success); }
.bz-bad { color: var(--color-error); }
.bz-note { font-size: var(--font-size-caption); color: var(--liki-text-2); margin: var(--space-2) 0 0; }

.bazhai-layout-panel[data-density='compact'] .bz-table { font-size: var(--font-size-micro); }
</style>
