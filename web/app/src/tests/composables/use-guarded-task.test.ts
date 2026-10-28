import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

const { useGuardedTask } = await import('@/composables/useGuardedTask')

describe('useGuardedTask', () => {
  function setup(report: (message: string) => void) {
    let task!: ReturnType<typeof useGuardedTask>
    mount(defineComponent({
      setup() {
        task = useGuardedTask({ report })
        return () => null
      },
    }))
    return task
  }

  it('toggles loading around the task', async () => {
    const task = setup(vi.fn())
    expect(task.loading.value).toBe(false)

    const promise = task.run(async () => {
      expect(task.loading.value).toBe(true)
    }, 'failed')
    expect(task.loading.value).toBe(true)
    await promise
    expect(task.loading.value).toBe(false)
  })

  it('reports thrown errors through the fallback message', async () => {
    const report = vi.fn()
    const task = setup(report)

    await task.run(async () => {
      throw new Error('boom')
    }, '回退消息')

    expect(report).toHaveBeenCalledOnce()
    expect(String(report.mock.calls[0][0])).toContain('回退消息')
  })

  it('does not report when the task handles a failure directly', async () => {
    const report = vi.fn()
    const task = setup(report)

    await task.run(async () => {
      report('domain failure')
    }, 'fallback')

    expect(report).toHaveBeenCalledExactlyOnceWith('domain failure')
  })

  it('guards against concurrent runs', async () => {
    const task = setup(vi.fn())
    let release!: () => void
    const gate = new Promise<void>(resolve => {
      release = resolve
    })
    const runs: string[] = []

    const first = task.run(async () => {
      runs.push('first')
      await gate
    }, 'failed')
    const second = task.run(async () => {
      runs.push('second')
    }, 'failed')
    release()
    await Promise.all([first, second])

    expect(runs).toEqual(['first'])
  })
})
