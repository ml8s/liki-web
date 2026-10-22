<template>
  <section>
    <AppPageHeader
      :title="$t('router.partnerCenter')"
      :subtitle="headerSubtitle"
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
      v-if="loading && !states"
      :rows="5"
      animated
      aria-busy="true"
    />
    <el-result
      v-else-if="error && !states"
      icon="error"
      :title="$t('common.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="load"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>

    <template v-else>
      <el-alert
        v-if="error"
        type="error"
        :title="error"
        show-icon
        :closable="false"
        class="load-alert"
      />

      <el-empty
        v-if="!visiblePrograms.length"
        :description="$t('partner.inviteOnly')"
      />

      <div
        v-else
        v-loading="loading"
        :aria-busy="loading"
      >
        <div class="role-grid">
          <el-card
            v-for="program in visiblePrograms"
            :key="program.role"
            shadow="never"
            class="role-card"
          >
            <template #header>
              <PartnerCardHeader>
                {{ program.title }}
                <template #actions>
                  <el-tag
                    size="small"
                    :type="program.tagType"
                  >
                    {{ program.stateLabel }}
                  </el-tag>
                </template>
              </PartnerCardHeader>
            </template>
            <p>{{ program.description }}</p>

            <el-alert
              v-if="program.state.application?.status === 'rejected'"
              :title="rejectionTitle(program.state.application.decision_reason)"
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
                v-if="program.state.status === 'active'"
                tag="router-link"
                :to="program.dashboard"
                type="primary"
              >
                {{ $t('partner.openWorkspace') }}
              </el-button>
              <el-button
                v-else-if="program.state.status === 'retired'"
                tag="router-link"
                :to="program.dashboard"
                type="primary"
              >
                {{ $t('partner.viewSettlement') }}
              </el-button>
              <el-button
                v-else-if="program.canApply"
                tag="router-link"
                :to="program.apply"
                type="primary"
              >
                {{ $t('partner.applyNow') }}
              </el-button>
            </div>
          </el-card>
        </div>
      </div>

      <el-card
        v-if="rules.length"
        shadow="never"
      >
        <template #header>
          <PartnerCardHeader>
            {{ $t('partner.rules') }}
          </PartnerCardHeader>
        </template>
        <ol class="rules-list">
          <li
            v-for="rule in rules"
            :key="rule"
          >
            {{ rule }}
          </li>
        </ol>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { errorMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import { fetchPartnerStates, type PartnerProgramState, type PartnerStates } from '@/api/partner'
import { partnerRoleState, type PartnerStatusTagType } from '@/utils/partnerStatus'

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
  tagType: PartnerStatusTagType
  canApply: boolean
}

// 我的合作：只显示本人已有关系的角色（审核中/已开通/结算/冻结/被拒），
// 绝不显示另一角色的任何入口——角色隔离由过滤保证，而非文案约定。
const visiblePrograms = computed<ProgramCard[]>(() => {
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
  ]
    .filter(program => Boolean(program.state.status) || program.state.application !== null)
    .map(program => {
      const other = program.role === 'kol' ? states.value?.scout : states.value?.kol
      const stateMeta = partnerRoleState(program.state, Boolean(other?.status))
      return {
        ...program,
        stateLabel: stateMeta.label,
        tagType: stateMeta.tag,
        canApply: program.state.can_submit,
      }
    })
})

const activeRole = computed<'kol' | 'scout' | null>(() =>
  visiblePrograms.value.length === 1 ? visiblePrograms.value[0].role : null,
)

const headerSubtitle = computed(() =>
  activeRole.value ? t(`partner.${activeRole.value}.subtitle`) : t('partner.inviteOnly'),
)

const rules = computed(() => {
  const role = activeRole.value
  if (!role) return []
  const cap = role === 'kol' ? 'Kol' : 'Scout'
  return [t(`partner.rule${cap}1`), t(`partner.rule${cap}2`), t('partner.rule3'), t('partner.rule4')]
})

function rejectionTitle(reason?: string) {
  return reason ? t('partner.rejectedTitle', { reason }) : t('partner.rejectedTitleNoReason')
}

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
.role-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: var(--space-5); margin-bottom: var(--space-5); }
.role-card p { margin: 0 0 var(--space-4); color: var(--liki-text-2); }
.card-alert, .load-alert { margin-bottom: var(--space-4); }
.actions { display: flex; justify-content: flex-end; }
.rules-list { margin: 0; padding-left: var(--space-5); color: var(--liki-text-2); line-height: 1.9; }
@media (max-width: 640px) { .role-grid { grid-template-columns: 1fr; } }
</style>
