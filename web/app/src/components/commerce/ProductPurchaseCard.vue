<template>
  <el-card
    class="app-card app-card--highlight product-card"
    shadow="never"
  >
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">{{ product.name }}</h2>
        <el-tag :type="product.purchased ? 'success' : 'warning'" size="small">
          {{ $t(product.purchased ? 'commerce.activated' : 'commerce.inactive') }}
        </el-tag>
      </div>
    </template>

    <p>{{ product.description }}</p>
    <ul class="skills">
      <li v-for="s in chatServices" :key="s.key">{{ $t(s.labelKey) }} · {{ $t(s.descKey) }}</li>
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
        {{ $t(product.purchased ? 'commerce.activated' : 'commerce.activateNow') }}
      </el-button>
    </div>
    <p class="disclaimer">{{ $t('commerce.aiDisclaimer') }}</p>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatProduct } from '@/api/commerce'
import { chatServices } from '@/config/chat'
const props = defineProps<{
  product: ChatProduct
  buying?: boolean
}>()

defineEmits<{ purchase: [] }>()

const { locale } = useI18n()

const priceLabel = computed(() => {
  return new Intl.NumberFormat(locale.value, {
    style: 'currency',
    currency: props.product.display_currency,
  }).format(props.product.display_amount_minor / 100)
})
</script>

<style scoped>
.product-card { margin-bottom: 20px; }
.header { display: flex; align-items: center; justify-content: space-between; }
p { color: var(--liki-text-2); margin: 0 0 12px; }
.skills { display: flex; flex-wrap: wrap; gap: 8px; padding-left: 18px; margin: 0 0 20px; color: var(--liki-text-2); font-size: 13px; }
.actions { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.price { font-size: 22px; font-weight: 700; color: var(--liki-ink); font-variant-numeric: tabular-nums; }
.disclaimer {
  margin: 14px 0 0;
  color: var(--liki-text-3);
  font-size: 12px;
  line-height: 1.5;
}
@media (max-width: 640px) {
  .actions { align-items: stretch; flex-direction: column; }
}
</style>
