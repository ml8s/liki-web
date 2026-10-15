<template>
  <!-- 全项目唯一 v-html 注入点：html 经 markdown.ts 的 DOMPurify 白名单消毒。 -->
  <div
    class="app-markdown"
    v-html="html"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderSafeMarkdown } from '@/utils/markdown'

const props = defineProps<{
  source: string
}>()

const html = computed(() => renderSafeMarkdown(props.source))
</script>

<style scoped>
.app-markdown {
  overflow-wrap: anywhere;
  line-height: 1.7;
  color: var(--liki-ink);
}
.app-markdown :deep(h1),
.app-markdown :deep(h2),
.app-markdown :deep(h3),
.app-markdown :deep(h4),
.app-markdown :deep(h5),
.app-markdown :deep(h6) {
  margin: 20px 0 10px;
  color: var(--liki-ink);
  font-weight: 650;
  line-height: 1.3;
  letter-spacing: -0.02em;
}
.app-markdown :deep(h1:first-child),
.app-markdown :deep(h2:first-child),
.app-markdown :deep(h3:first-child) {
  margin-top: 0;
}
.app-markdown :deep(h1) { font-size: var(--font-size-title-lg); }
.app-markdown :deep(h2) { font-size: var(--font-size-title-md); }
.app-markdown :deep(h3) { font-size: var(--font-size-title-xs); }
.app-markdown :deep(h4),
.app-markdown :deep(h5),
.app-markdown :deep(h6) { font-size: var(--font-size-title-sm); }
.app-markdown :deep(p) {
  margin: 0 0 12px;
  text-wrap: pretty;
}
.app-markdown :deep(p:last-child) { margin-bottom: 0; }
.app-markdown :deep(ul),
.app-markdown :deep(ol) {
  margin: 8px 0 12px;
  padding-left: 22px;
}
.app-markdown :deep(li) { margin: 4px 0; }
.app-markdown :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 14px;
  border-left: 3px solid var(--liki-primary-border);
  border-radius: 0 8px 8px 0;
  background: var(--liki-primary-tint);
  color: var(--liki-text-2);
}
.app-markdown :deep(blockquote p) { margin: 0; }
.app-markdown :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: var(--liki-paper);
  color: var(--liki-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.875em;
}
.app-markdown :deep(pre) {
  margin: 12px 0;
  padding: 14px;
  overflow-x: auto;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-control);
  background: var(--liki-paper);
}
.app-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
  font-size: var(--font-size-label);
  line-height: 1.6;
  white-space: pre;
}
.app-markdown :deep(table) {
  width: 100%;
  margin: 12px 0;
  border-collapse: collapse;
  font-size: var(--font-size-body);
}
.app-markdown :deep(th),
.app-markdown :deep(td) {
  padding: 8px 10px;
  border: 1px solid var(--liki-line);
  text-align: left;
}
.app-markdown :deep(th) {
  background: var(--liki-paper);
  color: var(--liki-text-2);
  font-weight: 600;
}
.app-markdown :deep(a) {
  color: var(--liki-primary);
  font-weight: 500;
  text-underline-offset: 3px;
}
.app-markdown :deep(hr) {
  margin: 18px 0;
  border: 0;
  border-top: 1px solid var(--liki-line);
}
</style>
