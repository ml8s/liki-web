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

.partner-stat {
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-paper);
}

.partner-stat dt {
  margin-bottom: var(--space-2);
  color: var(--liki-text-2);
  font-size: 13px;
}

.partner-stat dd {
  margin: 0;
  font-size: 20px;
  font-variant-numeric: tabular-nums;
}
</style>
