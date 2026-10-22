<template>
  <el-card
    class="section-card"
    shadow="never"
  >
    <template #header>
      <PartnerCardHeader>
        {{ title }}
        <template #actions>
          <el-button
            size="small"
            @click="$emit('copy')"
          >
            {{ $t('common.copy') }}
          </el-button>
        </template>
      </PartnerCardHeader>
    </template>
    <el-input
      :model-value="referralLink"
      :aria-label="$t('partner.referralLink')"
      :name="inputName"
      autocomplete="off"
      spellcheck="false"
      readonly
    />
    <slot />
    <PartnerStatGrid
      class="overview-stats"
      :stats="stats"
    />
  </el-card>
</template>

<script setup lang="ts">
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import PartnerStatGrid, { type PartnerStat } from '@/components/partner/PartnerStatGrid.vue'

defineProps<{
  title: string
  referralLink: string
  inputName: string
  stats: PartnerStat[]
}>()

defineEmits<{ copy: [] }>()
</script>
