import { describe, expect, it } from 'vitest'
import { huangliApiEventLabel } from '@/utils/huangliEvents'

describe('huangli engine event labels', () => {
  it('always uses the Simplified-Chinese engine contract', () => {
    expect(huangliApiEventLabel('wedding')).toBe('结婚')
    expect(huangliApiEventLabel('bed_install')).toBe('安床')
  })

  it('fails closed for an unknown event', () => {
    expect(huangliApiEventLabel('unknown-event')).toBe('unknown-event')
  })
})
