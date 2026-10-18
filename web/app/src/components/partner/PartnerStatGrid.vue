<template>
  <dl class="partner-stat-grid">
    <div
      v-for="stat in stats"
      :key="stat.key"
      class="partner-stat"
    >
      <dt>{{ stat.label }}</dt>
      <dd>{{ stat.value }}</dd>
    </div>
  </dl>
</template>

<script lang="ts">
export interface PartnerStat {
  key: string
  label: string
  value: string | number
}

</script>

<script setup lang="ts">
defineProps<{
  stats: PartnerStat[]
}>()
</script>

<style scoped>
.partner-stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-4);
  margin: 0;
}

/* 外卡圆角 --radius-card(16) - 卡内边距 --space-5(20) 为负，取小一级的同心圆角。 */
.partner-stat {
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-control);
  background: var(--liki-paper);
}

.partner-stat dt {
  margin-bottom: var(--space-2);
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}

.partner-stat dd {
  margin: 0;
  font-size: var(--font-size-title-md);
  font-variant-numeric: tabular-nums;
}
</style>
