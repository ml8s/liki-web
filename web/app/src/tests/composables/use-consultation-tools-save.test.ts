import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import type { BirthProfile } from '@/api/birth-profiles'

const saveBondRecord = vi.hoisted(() => vi.fn())

vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: () => ({ save: saveBondRecord }),
}))

const { useConsultationTools } = await import('@/composables/useConsultationTools')

const profile = {
  id: 'p1',
  nickname: '张三',
  birth: {
    gender: 'female',
    date: '1990-05-12',
    time: '08:30',
    instant: '1990-05-12T00:30:00Z',
    location: { longitude: 121.47, timezone: 'Asia/Shanghai' },
  },
} as unknown as BirthProfile

describe('useConsultationTools save outcome reporting', () => {
  function setup(handlers: { onError: (m: string) => void; onNotice?: (m: string) => void }) {
    let tools!: ReturnType<typeof useConsultationTools>
    mount(defineComponent({
      setup() {
        tools = useConsultationTools({
          selectedProfile: () => profile,
          personLabel: () => '张三',
          ...handlers,
        })
        return () => null
      },
    }))
    return tools
  }

  function primeHepan(tools: ReturnType<typeof useConsultationTools>) {
    tools.hepanInput.value = {
      secondBirth: {
        gender: 'male',
        date: '1992-01-01',
        time: '12:00',
        location: { longitude: 121.47, timezone: 'Asia/Shanghai' },
      },
      thirdPartyConsent: true,
    } as never
    tools.hepanResult.value = { bond: {} } as never
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    saveBondRecord.mockReset()
  })

  it('announces a successful explicit save through onNotice', async () => {
    saveBondRecord.mockResolvedValue(true)
    const onNotice = vi.fn()
    const onError = vi.fn()
    const tools = setup({ onError, onNotice })
    primeHepan(tools)

    tools.saveHepanRecord()
    await flushPromises()

    expect(onNotice).toHaveBeenCalledWith(expect.stringContaining('已保存'))
    expect(onError).not.toHaveBeenCalled()
  })

  it('surfaces a failed explicit save through onError', async () => {
    saveBondRecord.mockResolvedValue(false)
    const onNotice = vi.fn()
    const onError = vi.fn()
    const tools = setup({ onError, onNotice })
    primeHepan(tools)

    tools.saveHepanRecord()
    await flushPromises()

    expect(onError).toHaveBeenCalledWith(expect.stringContaining('保存失败'))
    expect(onNotice).not.toHaveBeenCalled()
  })
})
