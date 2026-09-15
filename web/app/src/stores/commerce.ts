import { defineStore } from 'pinia'
import router from '@/router'
import { checkoutChatProduct, fetchChatProducts, type ChatProduct } from '@/api/commerce'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useProfilesStore } from '@/stores/profiles'

type PaymentState = 'idle' | 'verifying' | 'confirmed' | 'unpaid'

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
    chatPurchased: () => useEntitlementsStore().allowedSkills.length === 4,
    showProductError: state => !!state.productsError && !state.products.length,
    initialLoading: state =>
      !state.products.length && (state.loadingProducts || useEntitlementsStore().loading || useChatStore().loadingSessions),
  },
  actions: {
    firstQuery(value: unknown): string {
      if (Array.isArray(value)) return String(value[0] ?? '')
      return String(value ?? '')
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
        this.productsError = error instanceof Error ? error.message : '加载产品失败'
      } finally {
        this.loadingProducts = false
      }
    },
    async buy() {
      const product = this.chatProduct
      if (!product) return
      this.buying = true
      this.actionError = ''
      try {
        const checkout = await checkoutChatProduct(product.id)
        window.location.href = checkout.checkout_url
      } catch (error) {
        this.actionError = error instanceof Error ? error.message : '发起支付失败'
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
          useChatStore().loadSessions(),
          this.loadProducts(),
        ])
      } finally {
        this.refreshing = false
      }
    },
    async clearPaymentQuery() {
      await router.replace({ path: '/chat', query: {} })
    },
    async handlePaymentReturn() {
      const query = router.currentRoute.value.query
      const orderId = this.firstQuery(query.order_id)
      const paymentIntentId = this.firstQuery(query.payment_intent_id)
      const status = this.firstQuery(query.status)
      if (!orderId && !paymentIntentId) return

      if (status === 'paid') {
        this.paymentState = 'verifying'
        this.paymentMessage = '支付已确认，正在开通 Liki Chat…'
        // Webhook 与 durable fulfillment 可能晚于浏览器返回，短暂轮询而非误报未开通。
        for (let attempt = 0; attempt < 12; attempt += 1) {
          await this.refresh()
          if (this.chatPurchased) {
            this.paymentState = 'confirmed'
            this.paymentMessage = '支付成功，Liki Chat 已开通。'
            await this.clearPaymentQuery()
            await this.enterInterpretation()
            return
          }
          await this.wait(1000 * (attempt + 1))
        }
        this.paymentState = 'verifying'
        this.paymentMessage = '支付已收到，权益开通仍在处理。请稍后刷新本页。'
        await this.clearPaymentQuery()
        return
      }

      if (status === 'unpaid') {
        this.paymentState = 'unpaid'
        this.paymentMessage = '支付尚未完成。如已完成付款，请稍后刷新订单状态。'
      }
      await this.clearPaymentQuery()
    },
    // V4：支付成功后直达解读——有档案则用默认 skill（八字）建会话并跳转。
    async enterInterpretation() {
      const profiles = useProfilesStore()
      if (!profiles.list.length) {
        await profiles.loadList()
      }
      const profileId = profiles.list[0]?.id ?? ''
      if (!profileId) return
      try {
        const session = await useChatStore().createSession(profileId, 'liki-bazi')
        await router.replace(`/chat/${session.id}`)
      } catch (error) {
        this.actionError = error instanceof Error ? error.message : '创建会话失败'
      }
    },
  },
})