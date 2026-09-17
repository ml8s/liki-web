<template>
  <el-card class="product-card" shadow="hover">
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
      <div class="prices">
        <span v-if="product.prices.CNY">¥{{ formatMinor(product.prices.CNY) }}</span>
        <span v-if="product.prices.USD">${{ formatMinor(product.prices.USD) }}</span>
      </div>
      <el-button
        type="primary"
        size="large"
        :disabled="product.purchased"
        :loading="buying"
        @click="$emit('purchase')"
      >
        购买 Liki Chat
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { ChatProduct } from '@/api/commerce'
import { chatServices } from '@/config/chat'

defineProps<{
  product: ChatProduct
  buying?: boolean
}>()

defineEmits<{ purchase: [] }>()

function formatMinor(amount: number): string {
  return (amount / 100).toFixed(2)
}
</script>

<style scoped>
.product-card { border-color: #f3d19e; }
.header { display: flex; align-items: center; justify-content: space-between; }
p { color: #555; }
.skills { display: flex; flex-wrap: wrap; gap: 8px; padding-left: 18px; margin: 12px 0 20px; color: var(--liki-text-2); }
.actions { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.prices { display: flex; gap: 14px; font-size: 20px; font-weight: 700; }
@media (max-width: 640px) {
  .actions { align-items: stretch; flex-direction: column; }
}
</style>
