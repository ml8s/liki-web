<template>
  <el-card
    class="app-card app-card--highlight product-card"
    shadow="never"
  >
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">
          {{ product.name }}
        </h2>
        <el-tag
          :type="product.purchased ? 'success' : 'warning'"
          size="small"
        >
          {{ $t(product.purchased ? 'commerce.activated' : 'commerce.inactive') }}
        </el-tag>
      </div>
    </template>

    <p>{{ $t('chat.productDescription') }}</p>
    <ul class="skills">
      <li
        v-for="s in chatIntents"
        :key="s.id"
      >
        {{ $t(s.labelKey) }} · {{ $t(s.descKey) }}
      </li>
    </ul>

    <div
      class="plans"
      :aria-busy="buying ? 'true' : 'false'"
    >
      <p
        v-if="!subscriptionOption && !oneTimeOption"
        class="plans-empty"
      >
        {{ $t('commerce.noPurchaseOptions') }}
      </p>
      <button
        v-if="subscriptionOption"
        type="button"
        class="plan plan--primary"
        :disabled="product.purchased || buying"
        @click="$emit('purchase', 'subscription')"
      >
        <span class="plan__name">{{ $t('commerce.subscription.title') }}</span>
        <span class="plan__price">{{ formatPrice(subscriptionOption.price) }}</span>
        <span class="plan__note">{{ $t('commerce.subscription.renewsMonthly') }}</span>
      </button>
      <button
        v-if="oneTimeOption"
        type="button"
        class="plan"
        :class="{ 'plan--primary': !subscriptionOption }"
        :disabled="product.purchased || buying"
        @click="$emit('purchase', 'one_time')"
      >
        <span class="plan__name">{{ $t('commerce.oneTime.title') }}</span>
        <span class="plan__price">{{ formatPrice(oneTimeOption.price) }}</span>
        <span class="plan__note">{{ $t('commerce.oneTime.access') }}</span>
      </button>
    </div>
    <p
      v-if="hasCnyOption"
      class="desktop-wechat-hint"
    >
      {{ $t('commerce.desktopWechatHint') }}
    </p>
    <p
      v-if="buying"
      class="plans-pending"
      role="status"
    >
      {{ $t('commerce.redirectingToCheckout') }}
    </p>
    <p class="disclaimer">
      {{ $t('commerce.aiDisclaimer') }}
    </p>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Product, MoneyMinor } from '@/api/sales'
import { chatIntents } from '@/config/chat'

const props = defineProps<{
  product: Product
  buying?: boolean
}>()

defineEmits<{ purchase: [billingMode: 'one_time' | 'subscription'] }>()

const { locale } = useI18n()

// 服务端只下发本区域可结算的选项（币种恒等于 display_currency）：国内是 Xunhu 的按次
// CNY，境外是 Dodo 的按次 + 订阅 USD。这里按计费模式取用，不再复刻区域规则。
const subscriptionOption = computed(() =>
  props.product.purchase_options?.find(option => option.billing_mode === 'subscription') ?? null,
)

const oneTimeOption = computed(() =>
  props.product.purchase_options?.find(option => option.billing_mode === 'one_time') ?? null,
)

const hasCnyOption = computed(() =>
  props.product.purchase_options?.some(option => option.price.currency_code === 'CNY') ?? false,
)

function formatPrice(price: MoneyMinor) {
  return new Intl.NumberFormat(locale.value, {
    style: 'currency',
    currency: price.currency_code,
  }).format(price.minor_units / 100)
}
</script>

<style scoped>
.product-card { margin-bottom: 20px; }
.header { display: flex; align-items: center; justify-content: space-between; }
p { color: var(--liki-text-2); margin: 0 0 12px; }
.skills { display: flex; flex-wrap: wrap; gap: 8px; padding-left: 18px; margin: 0 0 20px; color: var(--liki-text-2); font-size: var(--font-size-label); }
.plans { display: grid; gap: 12px; }
.plan {
  display: grid; gap: 4px; padding: 14px 16px; text-align: left; cursor: pointer;
  background: var(--liki-surface); border: 1px solid var(--liki-line); border-radius: var(--radius-lg);
  color: var(--liki-ink); font: inherit;
}
.plan--primary { border-color: var(--liki-primary); box-shadow: 0 0 0 1px var(--liki-primary) inset; }
.plan:disabled { opacity: .55; cursor: not-allowed; }
.plan__name { font-weight: 650; }
.plan__price { font-size: var(--font-size-title-md); font-weight: 750; font-variant-numeric: tabular-nums; }
.plan__note { color: var(--liki-text-3); font-size: var(--font-size-caption); }
.plans-empty { margin: 0; color: var(--liki-text-3); font-size: var(--font-size-label); }
.plans-pending { margin: 10px 0 0; color: var(--liki-text-3); font-size: var(--font-size-caption); }
.desktop-wechat-hint { display: none; margin: 10px 0 0; color: var(--liki-text-3); font-size: var(--font-size-caption); }
@media (min-width: 721px) {
  .desktop-wechat-hint { display: block; }
}
.disclaimer {
  margin: 14px 0 0;
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
  line-height: 1.5;
}
@media (max-width: 640px) {
  .plans { grid-template-columns: 1fr; }
}
</style>
