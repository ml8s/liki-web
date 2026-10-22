// 从 App 的 locale 目录抽取排盘共享组件所需的文案子树，按 locale 写入
// web/shared/i18n/chart-messages.<locale>.json（营销站 island 与 App 共同消费的拷贝）。
// 按 locale 拆文件 = 按加载边界组织：island-boot 动态 import 当前语言，首访不加载另两语。
// App 侧运行时仍用自己的完整 locale 文件；改动 App 文案后重跑本脚本即可保持同步。
// 同步校验见 check-chart-messages.js。
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const WEB_DIR = join(dirname(fileURLToPath(import.meta.url)), '..')
export const LOCALES = ['zh-Hans', 'zh-Hant', 'en']

// 共享组件用到的键：profiles.chart 整棵子树 + 风水（八宅/玄空）整棵子树 + 黄历择日整棵子树。
export function pickChartTree(source) {
  const tree = {
    profiles: {
      chart: source?.profiles?.chart ?? {},
      form: source?.profiles?.form ?? {},
      female: source?.profiles?.female ?? '',
      male: source?.profiles?.male ?? '',
    },
    services: {},
  }
  const fengshui = source?.services?.fengshui
  if (fengshui !== undefined) tree.services.fengshui = fengshui
  const huangli = source?.services?.huangli
  if (huangli !== undefined) tree.services.huangli = huangli
  const divination = source?.services?.divination
  if (divination !== undefined) tree.services.divination = divination
  const naming = source?.services?.naming
  if (naming !== undefined) tree.services.naming = naming
  const hepan = source?.services?.hepan
  if (hepan !== undefined) tree.services.hepan = hepan
  return tree
}

export function buildChartMessages() {
  const out = {}
  for (const locale of LOCALES) {
    const source = JSON.parse(
      readFileSync(join(WEB_DIR, 'app/src/locales', `${locale}.json`), 'utf-8'),
    )
    out[locale] = pickChartTree(source)
  }
  return out
}

export function chartMessagesPath(locale) {
  return join(WEB_DIR, 'shared/i18n', `chart-messages.${locale}.json`)
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const messages = buildChartMessages()
  mkdirSync(join(WEB_DIR, 'shared/i18n'), { recursive: true })
  for (const locale of LOCALES) {
    writeFileSync(chartMessagesPath(locale), `${JSON.stringify(messages[locale], null, 2)}\n`)
  }
  // 旧单文件（三语合体）退役：删除避免误用。
  const legacy = join(WEB_DIR, 'shared/i18n/chart-messages.json')
  rmSync(legacy, { force: true })
  console.log(`✓ chart-messages.<locale>.json written (${LOCALES.length} locales)`)
}
