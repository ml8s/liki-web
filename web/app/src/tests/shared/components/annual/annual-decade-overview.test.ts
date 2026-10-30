import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AnnualDecadeOverview from '@shared/components/annual/AnnualDecadeOverview.vue'
import { mergeAnnualDecade } from '@shared/domain/natal'

const decadeYears = mergeAnnualDecade(
  2027,
  Array.from({ length: 4 }, (_, index) => ({
    year: 2027 + index,
    nian_name: '丙午',
    shi_shen: '正官',
  })),
  Array.from({ length: 2 }, () => ({
    ming_gong: '午',
    ming_gong_name: '命宫',
    zhi: '午',
  })),
)

describe('AnnualDecadeOverview', () => {
  it('renders the requested decade and marks the selected year', async () => {
    const wrapper = mount(AnnualDecadeOverview, {
      props: { years: decadeYears, selectedYear: 2027 },
    })

    const yearCards = wrapper.findAll('.annual-year')
    expect(yearCards).toHaveLength(10)
    expect(yearCards[0]?.classes()).toContain('is-selected')
    expect(yearCards[0]?.text()).toContain('2027')
    await yearCards[2]?.trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([2029])
  })

  it('exposes missing and partial years as explicit states', () => {
    const wrapper = mount(AnnualDecadeOverview, {
      props: { years: decadeYears, selectedYear: 2027 },
    })

    expect(wrapper.findAll('[data-availability="ready"]')).toHaveLength(2)
    expect(wrapper.findAll('[data-availability="partial"]')).toHaveLength(2)
    expect(wrapper.findAll('[data-availability="missing"]')).toHaveLength(6)
  })
})
