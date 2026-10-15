import { expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import SkillCard from '@/components/chat/SkillCard.vue'

function viewFiles(directory = join(process.cwd(), 'src')): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(item => {
    const path = join(directory, item.name)
    if (item.isDirectory()) return item.name === 'views' || item.name === 'components' ? viewFiles(path) : []
    return item.name.endsWith('.vue') ? [path] : []
  })
}

it('does not render buttons nested inside router links', () => {
  const violations = viewFiles().flatMap(file => {
    const source = readFileSync(file, 'utf8')
    return /<router-link[\s\S]{0,400}?<el-button[\s\S]{0,400}?<\/router-link>/.test(source) ? [file] : []
  })
  expect(violations).toEqual([])
})

it('does not use an inert anchor tag for Element Plus route buttons', () => {
  const violations = viewFiles().flatMap(file => {
    const source = readFileSync(file, 'utf8')
    return /tag="a"/.test(source) ? [file] : []
  })
  expect(violations).toEqual([])
})

it('renders route buttons with a real href', async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/orders', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  const wrapper = mount(SkillCard, {
    props: {
      label: 'Liki Chat',
      description: 'Unlock the consultation.',
      enabled: false,
      actionTo: '/orders',
    },
    global: { plugins: [router] },
  })

  expect(wrapper.find('a').attributes('href')).toBe('/orders')
  expect(wrapper.find('a button').exists()).toBe(false)
})
