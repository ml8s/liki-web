import { defineStore } from 'pinia'
import { checkoutChatProduct, fetchChatProducts, type ChatProduct } from '@/api/commerce'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useProfilesStore } from '@/stores/profiles'
import { toUserMessage } from '@/utils/errors'
import { toFirstQueryString } from '@/utils/format'
import { i18n } from '@/i18n'

function t(key: string): string {
  return i18n.global.t(key)
}

type PaymentState = 'idle' | 'verifying' | 'confirmed' | 'unpaid'

/** handlePaymentReturn 返回值：由调用方决定路由跳转。 */
export interface PaymentReturnResult {
  action: 'none' | 'poll' | 'enter_chat' | 'show_message'
  sessionId?: string
  message?: string
  state?: PaymentState
}

interface CommerceState {
  products: ChatProduct[]
  productsError: string
  loadingProducts: boolean
  buying: boolean
  refreshing: boolean
  actionError: string
  paymentState: PaymentState
  paymentMessage: string
}

export const useCommerceStore = defineStore('commerce', {
  state: (): CommerceState => ({
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
        this.products = await fetchChatProducts()
      } catch (error) {
        this.productsError = toUserMessage(error, t('commerce.productsFailed'))
      } finally {
        this.loadingProducts = false
      }
    },
    async buy(billingMode: 'one_time' | 'subscription' = 'one_time') {
      const product = this.chatProduct
      if (!product) return
      this.buying = true
      this.actionError = ''
      try {
        const checkout = await checkoutChatProduct(product.id, billingMode, i18n.global.locale.value)
        // 只跟随相对路径或 https 支付域：防御后端配置错误导致的开放重定向。
        const url = checkout.checkout_url
        if (!url.startsWith('/') && !url.startsWith('https://')) {
          this.actionError = t('commerce.checkoutFailed')
          await this.refresh()
          return
        }
        window.location.href = url
      } catch (error) {
        this.actionError = toUserMessage(error, t('commerce.checkoutFailed'))
        await this.refresh()
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
      const orderId = toFirstQueryString(query.order_id)
      const paymentIntentId = toFirstQueryString(query.payment_intent_id)
      const status = toFirstQueryString(query.status)
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
            return { action: 'enter_chat' }
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
    // V4：支付成功后直达解读——有档案则用默认 skill（八字）建会话并跳转。
    async enterInterpretation(): Promise<string | null> {
      const profiles = useProfilesStore()
      if (!profiles.list.length) {
        await profiles.loadList()
      }
      const profileId = profiles.list[0]?.id ?? ''
      if (!profileId) return null
      try {
        const session = await useChatStore().createSession(profileId, 'liki')
        return session.id
      } catch (error) {
        this.actionError = toUserMessage(error, t('chat.createFailed'))
        return null
      }
    },
  },
})
