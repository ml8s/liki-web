<template>
  <div class="service-entries">
    <button
      v-for="item in items"
      :key="item.id"
      type="button"
      class="entry-row"
      :class="{ selected: item.id === selectedId }"
      @click="$emit('select', item.id)"
    >
      <span
        class="entry-glyph font-serif"
        aria-hidden="true"
      >{{ item.glyph }}</span>
      <span class="entry-copy">
        <span class="entry-name">{{ item.label }}</span>
        <span class="entry-desc">{{ item.desc }}</span>
      </span>
      <span
        v-if="item.action"
        class="entry-action"
        aria-hidden="true"
      >{{ item.action }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
// 场景入口行式列表（单一实现）：图标字 + 名称 + 一行说明 + 右侧动作。
// 三处共用——工作台「更多场景」、档案详情服务入口、Chat 领域选择。
export interface ServiceEntry {
  id: string
  glyph: string
  label: string
  desc: string
  /** 右侧动作文案（如「去排盤」「開始」）；缺省仅整行可点。 */
  action?: string
}

withDefaults(defineProps<{ items: ServiceEntry[]; selectedId?: string }>(), { selectedId: '' })
defineEmits<{ select: [id: string] }>()
</script>

<style scoped>
.service-entries {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-2);
}
.entry-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  min-height: 60px;
  padding: 10px 12px;
  text-align: left;
  font: inherit;
  color: inherit;
  cursor: pointer;
  background: var(--liki-surface);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
  transition: border-color var(--liki-dur) var(--liki-ease), background var(--liki-dur) var(--liki-ease);
}
.entry-row:hover,
.entry-row.selected {
  border-color: var(--liki-primary-border);
  background: var(--liki-paper);
}
.entry-row:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: 2px;
}
.entry-glyph {
  flex: none;
  display: inline-grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-size: var(--font-size-title-sm);
  font-weight: 600;
}
.entry-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.entry-name {
  font-weight: 600;
  font-size: var(--font-size-body);
  color: var(--liki-ink);
}
.entry-desc {
  font-size: var(--font-size-caption);
  color: var(--liki-text-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.entry-action {
  flex: none;
  font-size: var(--font-size-label);
  color: var(--liki-primary);
}
@media (max-width: 640px) {
  .service-entries { grid-template-columns: 1fr; }
  .entry-action { display: none; }
}
</style>
