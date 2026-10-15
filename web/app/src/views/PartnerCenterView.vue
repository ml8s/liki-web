<template>
  <section>
    <AppPageHeader
      :title="$t('router.partnerCenter')"
      :subtitle="$t('partner.centerSubtitle')"
    >
      <template #actions>
        <el-button
          :loading="loading"
          @click="load"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-skeleton
      v-if="loading"
      :rows="5"
      animated
      aria-busy="true"
    />
    <el-alert
      v-else-if="error"
      type="error"
      :title="error"
      show-icon
      :closable="false"
    />

    <div
      v-else
      class="role-grid"
    >
      <el-card
        v-for="program in programs"
        :key="program.role"
        shadow="never"
        class="role-card"
      >
        <template #header>
          <div class="role-head">
            <span class="role-name">{{ program.title }}</span>
            <el-tag
              size="small"
              :type="program.tagType"
            >
              {{ program.stateLabel }}
            </el-tag>
          </div>
        </template>
        <p>{{ program.description }}</p>

        <el-alert
          v-if="program.state.application?.status === 'rejected'"
          :title="$t('partner.rejectedTitle', { reason: program.state.application.decision_reason || $t('partner.defaultRejectionReason') })"
          type="warning"
          :closable="false"
          show-icon
          class="card-alert"
        />
        <el-alert
          v-else-if="program.state.status === 'frozen'"
          :title="$t('partner.currentRoleFrozen')"
          type="warning"
          :closable="false"
          show-icon
          class="card-alert"
        />

        <div class="actions">
          <el-button
            v-if="program.state.status === 'active' || program.state.status === 'retired'"
            tag="router-link"
            :to="program.dashboard"
            type="primary"
          >
            {{ $t('partner.openWorkspace') }}
          </el-button>
          <el-button
            v-else-if="program.canApply"
            tag="router-link"
            :to="program.apply"
            type="primary"
          >
            {{ $t('partner.applyNow') }}
          </el-button>
          <el-button
            v-else-if="program.otherAssigned"
            tag="router-link"
            :to="program.role === 'kol' ? '/partners/scout' : '/partners/kol'"
            link
          >
            {{ $t('partner.viewRoleDescription') }}
          </el-button>
        </div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>
        <span>{{ $t('partner.rules') }}</span>
      </template>
      <ol class="rules-list">
        <li>{{ $t('partner.rule1') }}</li>
        <li>{{ $t('partner.rule2') }}</li>
        <li>{{ $t('partner.rule3') }}</li>
        <li>{{ $t('partner.rule4') }}</li>
      </ol>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { errorMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import { fetchPartnerStates, type PartnerProgramState, type PartnerStates } from '@/api/partner'

const loading = ref(true)
const error = ref('')
const states = ref<PartnerStates | null>(null)
const { t } = useI18n()

type ProgramView = {
  role: 'kol' | 'scout'
  title: string
  description: string
  state: PartnerProgramState
  dashboard: string
  apply: string
}

type ProgramCard = ProgramView & {
  stateLabel: string
  tagType: 'success' | 'warning' | 'danger' | 'info'
  canApply: boolean
  otherAssigned: boolean
}

const programs = computed<ProgramCard[]>(() => {
  if (!states.value) return []
  return [
    {
      role: 'kol' as const, title: t('partner.kol.title'),
      description: t('partner.centerKolDescription'),
      state: states.value.kol, dashboard: '/partners/kol', apply: '/partners/kol',
    },
    {
      role: 'scout' as const, title: t('partner.scout.title'),
      description: t('partner.centerScoutDescription'),
      state: states.value.scout, dashboard: '/partners/scout', apply: '/partners/scout',
    },
  ].map(program => {
    const other = program.role === 'kol' ? states.value?.scout : states.value?.kol
    let stateLabel = t('partner.state.available')
    let tagType: 'success'|'warning'|'danger'|'info' = 'info'
    if (program.state.status === 'active') { stateLabel = t('partner.state.active'); tagType = 'success' }
    else if (program.state.status === 'retired') { stateLabel = t('partner.state.retired'); tagType = 'info' }
    else if (program.state.status === 'frozen') { stateLabel = t('partner.state.frozen'); tagType = 'danger' }
    else if (program.state.application?.status === 'pending') { stateLabel = t('partner.state.pending'); tagType = 'warning' }
    else if (program.state.application?.status === 'rejected') { stateLabel = t('partner.state.reapply'); tagType = 'warning' }
    else if (other?.status) { stateLabel = t('partner.state.conflict'); tagType = 'info' }
    return {
      ...program,
      stateLabel,
      tagType,
      canApply: program.state.can_submit,
      otherAssigned: Boolean(other?.status),
    }
  })
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    states.value = await fetchPartnerStates()
  } catch (err) {
    error.value = errorMessage(err) || t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.role-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 20px; }
.role-head { display: flex; align-items: center; justify-content: space-between; font-weight: 650; }
.role-name { font-size: 17px; }
.role-card p { min-height: 48px; margin: 0 0 14px; color: var(--liki-text-2); }
.card-alert { margin-bottom: 14px; }
.actions { display: flex; justify-content: flex-end; }
.rules-list { margin: 0; padding-left: 20px; color: var(--liki-text-2); line-height: 1.9; }
@media (max-width: 640px) { .role-grid { grid-template-columns: 1fr; } }
</style>
