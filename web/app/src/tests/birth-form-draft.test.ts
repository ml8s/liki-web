import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'
import BirthForm from '@/components/birth-profiles/BirthForm.vue'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

const draftKey = 'birth-profile:test'

beforeEach(() => {
  sessionStorage.clear()
})

async function mountForm(initial = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const drafts = useFormDraftsStore()
  drafts.set(draftKey, { nickname: 'Restored' })

  const wrapper = mount(BirthForm, {
    props: { initial, draftKey },
    global: {
      plugins: [pinia],
    },
  })
  return { wrapper, drafts }
}

describe('BirthForm draft integration', () => {
  it('restores saved values and marks the form dirty', async () => {
    const { wrapper } = await mountForm({ nickname: 'Server' })

    expect(wrapper.find('input').element.value).toBe('Restored')
    expect(wrapper.emitted('dirty')?.at(-1)).toEqual([true])
  })

  it('saves non-consent fields while dirty', async () => {
    const { wrapper, drafts } = await mountForm()
    await wrapper.find('input').setValue('Changed')

    const saved = drafts.get<{ nickname: string }>(draftKey)
    expect(saved?.nickname).toBe('Changed')
    expect(saved).not.toHaveProperty('third_party_consent')
  })
})
