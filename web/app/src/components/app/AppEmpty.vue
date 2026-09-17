<template>
  <div class="app-empty">
    <div class="empty-glyph font-serif">命</div>
    <p class="app-empty-desc">{{ description }}</p>
    <p class="app-empty-slogan font-serif">懂命理，用 Liki</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{ description: string }>()
</script>

<style scoped>
.app-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 48px 20px;
  text-align: center;
}
.empty-glyph {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  border: 1px solid var(--liki-gold);
  background: var(--liki-gold-tint);
  color: #8a6a2f;
  font-size: 28px;
  font-weight: 600;
  margin-bottom: 12px;
}
.app-empty-desc { margin: 0; font-size: 15px; color: var(--liki-ink); text-wrap: pretty; }
.app-empty-slogan { color: var(--liki-text-2); font-size: 13px; margin: 2px 0 8px; }
</style>