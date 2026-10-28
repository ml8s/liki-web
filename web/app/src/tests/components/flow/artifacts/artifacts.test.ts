import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ArtifactCapsule from '@/components/flow/artifacts/ArtifactCapsule.vue'
import ArtifactPanel from '@/components/flow/artifacts/ArtifactPanel.vue'
import ArtifactSurface from '@/components/flow/artifacts/ArtifactSurface.vue'
import { buildArtifactDescriptors, findArtifact } from '@/components/flow/artifacts/build'
import type { LiuyaoChartResult } from '@shared/api/engine/liuyao'
import type { ZiweiChart } from '@shared/api/engine/ziwei'

vi.mock('@vueuse/core', () => ({
  useMediaQuery: () => ({ value: true }),
}))

const chart: ZiweiChart = {
  gong_wei: [{
    index: '命宫',
    name: '命宫',
    xing_yao: [],
  }],
}

const ready = buildArtifactDescriptors('natal', {
  ziwei: { chart },
})

describe('artifact builder', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('creates descriptors only for supplied deterministic payloads', () => {
    expect(ready).toHaveLength(1)
    expect(ready[0]).toMatchObject({
      domainId: 'natal',
      kind: 'ziwei',
      state: 'ready',
    })
  })

  it('returns no fake artifacts when no payload exists', () => {
    expect(buildArtifactDescriptors('natal', {})).toEqual([])
  })

  it('finds a descriptor by id and returns null for unknown ids', () => {
    const descriptor = findArtifact(ready, 'natal:ziwei')
    expect(descriptor?.kind).toBe('ziwei')
    expect(findArtifact(ready, 'unknown')).toBeNull()
    expect(findArtifact(ready, null)).toBeNull()
  })
})

describe('ArtifactCapsule', () => {
  it('expands and exposes state to assistive technology', async () => {
    const wrapper = mount(ArtifactCapsule, {
      props: {
        descriptor: ready[0],
        open: false,
        surfaceId: 'surface-1',
      },
    })

    const button = wrapper.get('button')
    expect(button.attributes('aria-controls')).toBe('surface-1')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(button.text()).toContain('命理')
    expect(button.text()).toContain('可查看')

    await button.trigger('click')
    expect(wrapper.emitted('toggle')).toHaveLength(1)
  })
})

describe('ArtifactPanel', () => {
  it('renders the catalog panel for a ziwei chart', () => {
    const wrapper = mount(ArtifactPanel, {
      props: { descriptor: ready[0] },
      global: {
        stubs: {
          ZiweiChartPanel: { template: '<div data-testid="ziwei-chart" />' },
        },
      },
    })

    expect(wrapper.find('[data-testid="ziwei-chart"]').exists()).toBe(true)
  })

  it('uses the complete liuyao chart when the chart result exists', () => {
    const [descriptor] = buildArtifactDescriptors('divination', {
      liuyao: {
        casting: {
          mode: 'coins',
          order: 'bottom_up',
          yaos: [7, 7, 7, 7, 7, 7],
          dong_yao: [],
          casting_id: 'cast-test',
        },
        chart: {
          name: '乾为天',
          lines: [{}, {}, {}, {}, {}, {}],
          ben_gua: '乾为天',
          gong: '乾',
          gong_wuxing: '金',
          bian_yao: [{}, {}, {}, {}, {}, {}] as LiuyaoChartResult['bian_yao'],
          ri_chen_gan: '甲',
          ri_chen_zhi: '子',
          yue_jian_zhi: '子',
          yue_jian_gan: '甲',
          yong_shen: {},
          gua_ci: { name: '乾', gua_ci: '', yao_ci: ['', '', '', '', '', ''] },
          wang_shuai: ['', '', '', '', '', ''],
          ri_chen_relations: [{ relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }],
          xun_kong: ['戌', '亥'],
          casting: {
            mode: 'coins',
            order: 'bottom_up',
            yaos: [7, 7, 7, 7, 7, 7],
            dong_yao: [],
            casting_id: 'cast-test',
          },
        } satisfies LiuyaoChartResult,
      },
    })

    const wrapper = mount(ArtifactPanel, {
      props: { descriptor: descriptor! },
      global: {
        stubs: {
          LiuyaoChartPanel: { template: '<div data-testid="liuyao-chart" />' },
        },
      },
    })

    expect(wrapper.find('[data-testid="liuyao-chart"]').exists()).toBe(true)
  })

  it('falls back from a liuyao chart to its casting receipt', () => {
    const [descriptor] = buildArtifactDescriptors('divination', {
      liuyao: {
        casting: {
          mode: 'coins',
          order: 'bottom_up',
          yaos: [7, 7, 7, 7, 7, 7],
          dong_yao: [],
          casting_id: 'cast-test',
        },
        chart: null,
      },
    })

    const wrapper = mount(ArtifactPanel, {
      props: { descriptor: descriptor! },
      global: {
        stubs: {
          LiuyaoCastingPanel: { template: '<div data-testid="liuyao-casting" />' },
        },
      },
    })

    expect(wrapper.find('[data-testid="liuyao-casting"]').exists()).toBe(true)
  })

  it('renders an explicit unavailable state instead of fake data', () => {
    const wrapper = mount(ArtifactPanel, {
      props: {
        descriptor: {
          id: 'natal:ziwei',
          domainId: 'natal',
          kind: 'ziwei',
          titleKey: 'services.natal.label',
          state: 'unavailable',
        },
      },
    })

    expect(wrapper.text()).toContain('暂不可用')
  })
})

describe('ArtifactSurface', () => {
  it('uses a desktop sidecar when the media query matches', async () => {
    const wrapper = mount(ArtifactSurface, {
      props: {
        descriptor: ready[0],
        open: true,
        mode: 'pinned',
      },
    })
    await flushPromises()

    expect(wrapper.find('aside.artifact-sidecar--pinned').exists()).toBe(true)
    expect(wrapper.find('.artifact-sidecar__title').text()).toBe('命理')
  })

  it('keeps a closed desktop sidecar hidden but addressable', () => {
    const wrapper = mount(ArtifactSurface, {
      props: {
        descriptor: ready[0],
        open: false,
      },
    })

    expect(wrapper.find('aside.artifact-sidecar').exists()).toBe(true)
    expect(wrapper.find('aside.artifact-sidecar').isVisible()).toBe(false)
  })
})
