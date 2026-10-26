import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

const createRecord = vi.hoisted(() => vi.fn())

vi.mock('@/api/records', () => ({
  createRecord,
}))

const { useToolRecords } = await import('@/composables/useToolRecords')

describe('useToolRecords.save', () => {
  function setup() {
    let tools!: ReturnType<typeof useToolRecords>
    mount(defineComponent({
      setup() {
        tools = useToolRecords('bond')
        return () => null
      },
    }))
    return tools
  }

  beforeEach(() => {
    createRecord.mockReset()
  })

  it('reports success when the record is created', async () => {
    createRecord.mockResolvedValue({ id: 'r1' })
    const tools = setup()

    await expect(tools.save('title', { a: 1 })).resolves.toBe(true)
    expect(tools.error.value).toBe('')
    expect(tools.records.value).toHaveLength(1)
  })

  it('reports failure and keeps the list intact on error', async () => {
    createRecord.mockRejectedValue(new Error('offline'))
    const tools = setup()

    await expect(tools.save('title', { a: 1 })).resolves.toBe(false)
    expect(tools.error.value).toBeTruthy()
    expect(tools.records.value).toHaveLength(0)
  })
})
