import { beforeEach, describe, expect, it, vi } from 'vitest'

const reportAppFlowEvent = vi.hoisted(() => vi.fn())

vi.mock('@shared/utils/app-flow-telemetry', () => ({
  reportAppFlowEvent,
}))

const { useConsultationTelemetry } = await import('@/composables/useConsultationTelemetry')

describe('consultation telemetry', () => {
  const telemetry = useConsultationTelemetry()

  beforeEach(() => {
    reportAppFlowEvent.mockClear()
  })

  it('reports entry and restore without person identifiers', () => {
    telemetry.trackEntry({
      groupId: 'natal',
      seed: 'liunian',
      source: 'legacy_redirect',
      restorable: true,
    })
    telemetry.trackRestore({
      groupId: 'divination',
      source: 'history',
      restored: true,
    })

    expect(reportAppFlowEvent).toHaveBeenNthCalledWith(1, {
      event: 'consultation_entry',
      intentGroup: 'natal',
      source: 'legacy_redirect',
      intent: 'liunian',
      status: 'restorable',
    })
    expect(reportAppFlowEvent).toHaveBeenNthCalledWith(2, {
      event: 'consultation_restore',
      intentGroup: 'divination',
      source: 'history',
      status: 'restored',
    })
  })

  it('projects bridge results while preserving pending-backend huangli', () => {
    telemetry.trackBridge({
      groupId: 'natal',
      seed: 'huangli',
      result: {
        status: 'pending-backend',
        groupId: 'natal',
        pendingIntent: 'huangli',
      },
    })

    expect(reportAppFlowEvent).toHaveBeenCalledWith({
      event: 'consultation_bridge',
      intentGroup: 'natal',
      intent: 'huangli',
      status: 'pending_backend',
    })
  })
})
