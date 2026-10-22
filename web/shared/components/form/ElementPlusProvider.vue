<script setup lang="ts">
// Astro prerenders Vue islands; Element Plus needs a deterministic id seed on both server and client.
import { inject, provide } from 'vue'
import { ID_INJECTION_KEY, ZINDEX_INJECTION_KEY } from 'element-plus'
import '@shared/styles/element-plus/index.css'

// App 已有根 Provider 时保持其递增状态；Astro island 没有根 Provider 时才注入本地 seed。
if (!inject(ID_INJECTION_KEY, undefined)) {
  provide(ID_INJECTION_KEY, { prefix: 1024, current: 0 })
}
if (!inject(ZINDEX_INJECTION_KEY, undefined)) {
  provide(ZINDEX_INJECTION_KEY, { current: 0 })
}
</script>

<template>
  <slot />
</template>
