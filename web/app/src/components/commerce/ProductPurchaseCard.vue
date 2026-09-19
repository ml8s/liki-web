<template>
  <el-card class="product-card" shadow="never">
    <template #header>
      <div class="header">
        <strong>{{ product.name }}</strong>
        <el-tag :type="product.purchased ? 'success' : 'warning'" size="small">
          {{ product.purchased ? '已开通' : '未开通' }}
        </el-tag>
      </div>
    </template>

    <p>{{ product.description }}</p>
    <ul class="skills">
      <li v-for="s in chatServices" :key="s.key">{{ s.label }} · {{ s.desc }}</li>
    </ul>

    <div class="actions">
      <div class="price">{{ priceLabel }}</div>
      <el-button
        type="primary"
        size="large"
        :disabled="product.purchased"
        :loading="buying"
        @click="$emit('purchase')"
      >
        {{ product.purchased ? '已开通' : '立即开通' }}
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ChatProduct } from '@/api/commerce'
import { chatServices } from '@/config/chat'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  product: ChatProduct
  buying?: boolean
}>()

defineEmits<{ purchase: [] }>()

const auth = useAuthStore()

const isZh = computed(() => auth.user?.locale?.startsWith('zh') ?? true)

const priceLabel = computed(() => {
  if (isZh.value && props.product.prices.CNY) {
    return `¥${(props.product.prices.CNY / 100).toFixed(2)}`
  }
  if (props.product.prices.USD) {
    return `$${(props.product.prices.USD / 100).toFixed(2)}`
  }
  const minor = Object.values(props.product.prices)[0]
  return minor != null ? `${minor / 100}` : ''
})
</script>

<style scoped>
.product-card { border-radius: 12px; border-color: var(--liki-gold); margin-bottom: 20px; }
.header { display: flex; align-items: center; justify-content: space-between; }
p { color: var(--liki-text-2); margin: 0 0 12px; }
.skills { display: flex; flex-wrap: wrap; gap: 8px; padding-left: 18px; margin: 0 0 20px; color: var(--liki-text-2); font-size: 13px; }
.actions { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.price { font-size: 22px; font-weight: 700; color: var(--liki-ink); font-variant-numeric: tabular-nums; }
@media (max-width: 640px) {
  .actions { align-items: stretch; flex-direction: column; }
}
</style>
