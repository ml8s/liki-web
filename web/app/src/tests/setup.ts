import { config } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { i18n } from '@/i18n'
import { setAppLocale } from '@/i18n'

config.global.plugins = [ElementPlus, i18n]
await setAppLocale('zh-Hans', { persist: false })
