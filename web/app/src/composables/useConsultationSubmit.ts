import { ref, type MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatSession } from '@/api/chat'
import type { ConsultationContext } from '@shared/domain/consultation'
import { useChatStore } from '@/stores/chat'
import { useIntentFlowStore } from '@/stores/intent-flow'
import { useConsultationTelemetry } from '@/composables/useConsultationTelemetry'
import { classifyResourceFailure } from '@/utils/resourceStatus'
import { toUserMessage } from '@/utils/errors'

export interface ConsultationSubmitOptions {
  /** Sessions are matched only within this person's workspace. */
  personId: MaybeRefOrGetter<string | null>
  /** Called with a user-facing failure message; the view decides placement. */
  onError: (message: string) => void
  /** Called when the composer draft must be announced as preserved. */
  onDraftPreserved: () => void
  /** Called when a failed send restored the draft into the composer. */
  onDraftRestored: () => void
  /** Called after every send settles (success or handled failure); the view
   *  owns scroll positioning. */
  onAfterSend?: () => Promise<void> | void
}

export type SubmitSource = 'composer' | 'focused_form' | 'casting'

export interface FocusedFormSubmit {
  /** Intent skill that owns the context intent. */
  skill: string
  context: ConsultationContext
  /** True when an existing session already carries the identical context. */
  matchesContext: (session: ChatSession) => boolean
  content: string
  source?: SubmitSource
}

/**
 * Owns consultation session routing for a submit: reuse a matching durable
 * context session when one exists, otherwise create one, then send through the
 * active session. The view keeps validation and focused-form guards; this
 * composable guarantees every send is tracked and draft-safe.
 */
export function useConsultationSubmit(options: ConsultationSubmitOptions) {
  const { t } = useI18n()
  const chat = useChatStore()
  const flow = useIntentFlowStore()
  const telemetry = useConsultationTelemetry()

  const isSubmitting = ref(false)

  async function openMatchingAndSend(
    matcher: (session: ChatSession) => boolean,
    content: string,
    source: SubmitSource,
  ): Promise<boolean> {
    const personId = toValue(options.personId)
    const matchingSession = chat.sessions
      .filter(session => (session.profile_id || null) === personId)
      .find(matcher)
    if (!matchingSession) return false
    if (chat.activeSession?.id !== matchingSession.id) await chat.openSession(matchingSession.id)
    telemetry.trackSubmit({
      groupId: flow.groupId,
      source,
      created: false,
    })
    await sendUsingActiveSession(content)
    return true
  }

  async function createAndSendWithSession(
    skill: string,
    context: ConsultationContext | undefined,
    content: string,
    source: SubmitSource,
  ) {
    try {
      const session = context === undefined
        ? await chat.createSession(flow.personId ?? '', skill)
        : await chat.createSession(flow.personId ?? '', skill, context)
      chat.activeSession = session
      chat.messages = []
      chat.messagesCursor = ''
      telemetry.trackSubmit({
        groupId: flow.groupId,
        source,
        created: true,
      })
      await sendUsingActiveSession(content)
    } catch (error) {
      options.onError(toUserMessage(error, t('chat.createFailed')))
      options.onDraftPreserved()
      telemetry.trackFailure(flow.groupId, classifyResourceFailure(error))
    }
  }

  async function sendUsingActiveSession(content: string) {
    if (!chat.activeSession) return
    const draft = content
    flow.setDraft('')
    await chat.send(draft)
    if (chat.error) {
      flow.setDraft(draft)
      options.onDraftRestored()
    }
    await options.onAfterSend?.()
  }

  /**
   * One canonical focused-form path: reuse the session whose durable context
   * matches exactly, otherwise create a session with the projected context.
   */
  async function submitFocusedForm(submit: FocusedFormSubmit) {
    const source = submit.source ?? 'focused_form'
    const sent = await openMatchingAndSend(submit.matchesContext, submit.content, source)
    if (sent) return
    await createAndSendWithSession(submit.skill, submit.context, submit.content, source)
  }

  async function run(callback: () => Promise<void>) {
    if (isSubmitting.value) return
    isSubmitting.value = true
    try {
      await callback()
    } finally {
      isSubmitting.value = false
    }
  }

  return {
    isSubmitting,
    run,
    submitFocusedForm,
    openMatchingAndSend,
    createAndSendWithSession,
  }
}
