<template>
  <div class="page-stack">
    <AppPageHeader
      :title="$t('nav.mine')"
      :subtitle="$t('mine.subtitle')"
    >
      <template #actions>
        <AppExperienceSwitcher
          target="consultation"
          variant="topbar"
        >
          {{ $t('nav.consultation') }}
        </AppExperienceSwitcher>
      </template>
    </AppPageHeader>

    <ServiceEntryList
      :items="items"
      @select="open"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppExperienceSwitcher from '@/components/app/AppExperienceSwitcher.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import ServiceEntryList, { type ServiceEntry } from '@/components/app/ServiceEntryList.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()

const items = computed<ServiceEntry[]>(() => {
  const entries: ServiceEntry[] = [
    {
      id: 'profiles',
      glyph: '档',
      label: t('router.profiles'),
      desc: t('profiles.list.subtitle'),
    },
    {
      id: 'history',
      glyph: '史',
      label: t('router.chat'),
      desc: t('chat.subtitle'),
    },
    {
      id: 'orders',
      glyph: '订',
      label: t('router.orders'),
      desc: t('orders.subtitle'),
    },
    {
      id: 'account',
      glyph: '账',
      label: t('router.account'),
      desc: t('account.subtitle'),
    },
  ]

  if (auth.hasPartnerRole) {
    entries.push({
      id: 'partners',
      glyph: '合',
      label: t('router.partnerCenter'),
      desc: t('partner.mineDescription'),
    })
  }

  return entries
})

const routeNames = {
  profiles: 'birth-profiles',
  history: 'chat',
  orders: 'orders',
  account: 'account',
  partners: 'partner-center',
} as const

function open(id: string) {
  const name = routeNames[id as keyof typeof routeNames]
  if (name) void router.push({ name })
}
</script>
