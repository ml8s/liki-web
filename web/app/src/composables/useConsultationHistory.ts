import { computed, type MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatSession } from '@/api/chat'
import { useChatStore } from '@/stores/chat'
import { useDomainFlowStore } from '@/stores/domain-flow'
import { domainForSession } from '@shared/domain/flow'
import { confirmDeleteEntity, promptRenameEntity } from '@/utils/dialogs'
import { toUserMessage } from '@/utils/errors'
import { useConsultationTelemetry } from '@/composables/useConsultationTelemetry'

export interface ConsultationHistoryOptions {
  /** Sessions stay bound to the visible (person, domain) workspace. */
  personId: MaybeRefOrGetter<string | null>
  domainId: MaybeRefOrGetter<string>
}

/**
 * Owns the visible-session list and its actions for the consultation view.
 * Filters by the same (person, domain) binding the backend enforces, sorts by
 * recency, and centralises restore/rename/remove telemetry and error surface.
 */
export function useConsultationHistory(options: ConsultationHistoryOptions) {
  const { t } = useI18n()
  const chat = useChatStore()
  const flow = useDomainFlowStore()
  const telemetry = useConsultationTelemetry()

  const activeSessions = computed(() =>
    chat.sessions
      .filter(session => session.status === 'active')
      .sort((left, right) => right.updated_at.localeCompare(left.updated_at)),
  )

  const visibleSessions = computed(() =>
    chat.sessions
      .filter(session => (session.profile_id || null) === toValue(options.personId))
      .filter(session => domainForSession(session) === toValue(options.domainId))
      .sort((left, right) => right.updated_at.localeCompare(left.updated_at)),
  )

  const latestVisibleSession = computed(() =>
    activeSessions.value.find(session =>
      (session.profile_id || null) === toValue(options.personId)
      && domainForSession(session) === toValue(options.domainId),
    ) ?? null,
  )

  const restoreVisible = computed(() =>
    Boolean(latestVisibleSession.value) &&
    chat.activeSession?.id !== latestVisibleSession.value?.id &&
    !chat.loadingMessages,
  )

  async function restoreLatest() {
    const latest = latestVisibleSession.value
    if (!latest) {
      chat.resetActive()
      return
    }
    await chat.openSession(latest.id)
    telemetry.trackRestore({
      domainId: flow.domainId,
      source: 'latest',
      restored: chat.activeSession?.id === latest.id,
    })
  }

  /**
   * `null` intentionally means "across every person/domain" for boot-time
   * session-first restore. Omitting the argument uses the visible domain.
   */
  function latestRestorableSession(domainId: string | null = toValue(options.domainId)): ChatSession | null {
    if (domainId === null) return activeSessions.value[0] ?? null
    return activeSessions.value.find(session => domainForSession(session) === domainId) ?? null
  }

  async function openSession(session: ChatSession) {
    if (chat.activeSession?.id === session.id) return
    await chat.openSession(session.id)
    telemetry.trackRestore({
      domainId: flow.domainId,
      source: 'history',
      restored: chat.activeSession?.id === session.id,
    })
  }

  async function renameSession(session: ChatSession) {
    let value: string | null
    try {
      value = await promptRenameEntity(t('chat.sessions.noun'), session.title)
    } catch (error) {
      chat.error = toUserMessage(error, t('flow.historyActionFailed'))
      return
    }
    if (value === null) return
    try {
      await chat.renameSession(session.id, value)
    } catch (error) {
      chat.error = toUserMessage(error, t('flow.historyActionFailed'))
    }
  }

  async function removeSession(session: ChatSession) {
    const confirmed = await confirmDeleteEntity(t('chat.sessions.noun'), session.title)
    if (!confirmed) return
    try {
      await chat.removeSession(session.id)
    } catch (error) {
      chat.error = toUserMessage(error, t('flow.historyActionFailed'))
    }
  }

  return {
    visibleSessions,
    latestVisibleSession,
    latestRestorableSession,
    restoreVisible,
    restoreLatest,
    openSession,
    renameSession,
    removeSession,
  }
}
