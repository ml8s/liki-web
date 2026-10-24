import { defineStore } from 'pinia'
import { checkoutProduct, fetchProducts, type Product } from '@/api/sales'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { toUserMessage } from '@/utils/errors'
import { firstQueryValue } from '@/utils/query'
import { i18n } from '@/i18n'
import {
  consultationBirthFromFields,
  type ConsultationContext,
  type ConsultationScenario,
} from '@shared/domain/consultation'
import type { BirthFieldsValue } from '@shared/components/form/birth-fields'
import { consumeConsultationHandoff } from '@shared/utils/handoff'

function t(key: string): string {
  return i18n.global.t(key)
}

type PaymentState = 'idle' | 'verifying' | 'confirmed' | 'unpaid'

/** handlePaymentReturn 返回值：由调用方决定路由跳转。 */
export interface PaymentReturnResult {
  action: 'none' | 'enter_chat' | 'show_message'
  sessionId?: string
  scenarioId?: ConsultationScenario
  message?: string
  state?: PaymentState
}

const PAYMENT_SCENARIO_KEY = 'liki.pending-payment-scenario'
const consultationScenarios: ConsultationScenario[] = [
  'natal', 'liunian', 'hepan', 'divination', 'huangli', 'fengshui', 'naming',
]

function savePaymentScenario(scenario: string): void {
  try {
    if (consultationScenarios.includes(scenario as ConsultationScenario)) {
      sessionStorage.setItem(PAYMENT_SCENARIO_KEY, scenario)
    }
  } catch {
    // Storage may be unavailable; the chat default remains natal.
  }
}

function takePaymentScenario(): ConsultationScenario {
  try {
    const value = sessionStorage.getItem(PAYMENT_SCENARIO_KEY)
    sessionStorage.removeItem(PAYMENT_SCENARIO_KEY)
    if (value && consultationScenarios.includes(value as ConsultationScenario)) {
      return value as ConsultationScenario
    }
  } catch {
    // Fall through to the default.
  }
  return 'natal'
}

interface SalesState {
  products: Product[]
  productsError: string
  loadingProducts: boolean
  buying: boolean
  refreshing: boolean
  actionError: string
  paymentState: PaymentState
  paymentMessage: string
}

export const useSalesStore = defineStore('sales', {
  state: (): SalesState => ({
    products: [],
    productsError: '',
    loadingProducts: false,
    buying: false,
    refreshing: false,
    actionError: '',
    paymentState: 'idle',
    paymentMessage: '',
  }),
  getters: {
    chatProduct: state => state.products.find(item => item.id === 'liki-chat') ?? null,
    chatPurchased: () => useEntitlementsStore().purchased,
    showProductError: state => !!state.productsError && !state.products.length,
    initialLoading: state =>
      !state.products.length && (state.loadingProducts || useEntitlementsStore().loading || useChatStore().loadingSessions),
  },
  actions: {
    reset() {
      this.products = []
      this.productsError = ''
      this.loadingProducts = false
      this.buying = false
      this.refreshing = false
      this.actionError = ''
      this.paymentState = 'idle'
      this.paymentMessage = ''
    },
    wait(ms: number) {
      return new Promise(resolve => setTimeout(resolve, ms))
    },
    async loadProducts() {
      this.loadingProducts = true
      this.productsError = ''
      try {
        this.products = await fetchProducts()
      } catch (error) {
        this.productsError = toUserMessage(error, t('commerce.productsFailed'))
      } finally {
        this.loadingProducts = false
      }
    },
    /**
     * 支付失败后先同步远端状态，再写错误文案。refresh() 会清空 actionError，
     * 反过来写（先写错误再 refresh）等于把错误吞掉：买家只看到按钮闪一下，
     * 既没有报错也没有支付页，就是「点了没反应」。
     */
    async failCheckout(error: unknown, fallbackKey: string) {
      const message = toUserMessage(error, t(fallbackKey))
      try {
        await this.refresh()
      } catch {
        // 刷新失败不该盖掉支付失败的提示。
      }
      this.actionError = message
    },
    async buy(
      billingMode: 'one_time' | 'subscription' = 'one_time',
      scenarioId: ConsultationScenario | string = 'natal',
    ) {
      const product = this.chatProduct
      if (!product || this.buying) return
      this.buying = true
      this.actionError = ''
      savePaymentScenario(scenarioId)
      try {
        const checkout = await checkoutProduct(product.id, billingMode, i18n.global.locale.value)
        // 只跟随相对路径或 https 支付域：防御后端配置错误导致的开放重定向。
        const url = checkout.checkout_url
        if (!url.startsWith('/') && !url.startsWith('https://')) {
          await this.failCheckout(new Error('untrusted checkout url'), 'commerce.checkoutFailed')
          return
        }
        window.location.href = url
      } catch (error) {
        await this.failCheckout(error, 'commerce.checkoutFailed')
      } finally {
        this.buying = false
      }
    },
    async refresh() {
      this.refreshing = true
      this.actionError = ''
      try {
        await Promise.all([
          useEntitlementsStore().load(true),
          useChatStore().loadSessions(true),
          this.loadProducts(),
        ])
      } finally {
        this.refreshing = false
      }
    },
    /**
     * 处理支付回调。接收路由查询参数，返回结果对象供调用方决定路由跳转。
     * store 不再直接操作 router。
     */
    async handlePaymentReturn(query: Record<string, unknown>): Promise<PaymentReturnResult> {
      const orderId = firstQueryValue(query.order_id)
      const paymentIntentId = firstQueryValue(query.payment_intent_id)
      const status = firstQueryValue(query.status)
      if (!orderId && !paymentIntentId) return { action: 'none' }

      if (status === 'paid') {
        this.paymentState = 'verifying'
        this.paymentMessage = t('commerce.payment.confirmed')
        // Webhook 与 durable fulfillment 可能晚于浏览器返回，短暂轮询而非误报未开通。
        for (let attempt = 0; attempt < 12; attempt += 1) {
          await this.refresh()
          if (this.chatPurchased) {
            this.paymentState = 'confirmed'
            this.paymentMessage = t('commerce.payment.active')
            // 轮询成功后由调用方 enterInterpretation
            return { action: 'enter_chat', scenarioId: takePaymentScenario() }
          }
          const delay = Math.min(1000 * Math.pow(2, attempt), 5000)
          await this.wait(delay)
        }
        this.paymentState = 'verifying'
        this.paymentMessage = t('commerce.payment.processing')
        return { action: 'show_message', message: this.paymentMessage, state: 'verifying' }
      }

      if (status === 'unpaid') {
        this.paymentState = 'unpaid'
        this.paymentMessage = t('commerce.payment.pending')
      }
      return { action: 'show_message', message: this.paymentMessage, state: this.paymentState }
    },
    // 支付成功后直达解读。优先使用用户当前选择的档案；没有选择时才退回第一个档案。
    // scenarioId 保留用户在支付前选择的场景，避免所有支付都落到 natal（或错误的专家）。
    async enterInterpretation(
      preferredProfileId?: string,
      scenarioId: ConsultationScenario = 'natal',
    ): Promise<string | null> {
      const profiles = useBirthProfilesStore()
      if (!profiles.list.length) {
        await profiles.loadList()
      }
      const profileId = preferredProfileId || profiles.list[0]?.id || ''
      if (!profileId) return null
      try {
        const profile = profiles.list.find(item => item.id === profileId)
        // Tool handoffs carry the facts the user already computed (for example
        // a divination casting). Consume it once; otherwise build the minimal
        // profile-derived context for birth-based scenarios.
        const handoff = consumeConsultationHandoff<ConsultationContext>(scenarioId)
        const context = handoff ?? (profile ? consultationContextFor(profile, scenarioId) : null)
        const session = await useChatStore().createSession(profileId, scenarioId, context ?? undefined)
        return session.id
      } catch (error) {
        this.actionError = toUserMessage(error, t('chat.createFailed'))
        return null
      }
    },
  },
})

function consultationContextFor(profile: {
  birth: {
    gender: 'female' | 'male'
    date: string | null
    time: string | null
    location: { longitude: number | null; timezone: string | null }
  }
}, scenarioId: ConsultationScenario): ConsultationContext | null {
  if (!['natal', 'liunian', 'naming'].includes(scenarioId)) return null
  const { gender, date, time, location } = profile.birth
  if (!date || !time || location.longitude == null || !location.timezone) return null
  const birth: BirthFieldsValue = {
    gender,
    date,
    time,
    location: {
      place: '',
      longitude: location.longitude,
      timezone: location.timezone,
      source: 'saved',
    },
  }
  const birthFacts = consultationBirthFromFields(birth)
  if (scenarioId === 'liunian') {
    return {
      version: 1,
      scenario: 'liunian',
      birth: birthFacts,
      year: new Date().getFullYear(),
    }
  }
  return {
    version: 1,
    scenario: scenarioId,
    birth: birthFacts,
  }
}
