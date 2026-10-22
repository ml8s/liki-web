// 校验 web/shared/i18n/chart-messages.<locale>.json 与 App locale 目录一致（防漂移）。
// 不同步时：运行 node scripts/build-chart-messages.js 并提交产物。
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { buildChartMessages, chartMessagesPath, LOCALES } from './build-chart-messages.js'

const WEB_DIR = join(dirname(fileURLToPath(import.meta.url)), '..')
const messages = buildChartMessages()

for (const locale of LOCALES) {
  const target = chartMessagesPath(locale)
  if (!existsSync(target)) {
    console.error(`✗ 缺少 shared/i18n/chart-messages.${locale}.json：运行 node scripts/build-chart-messages.js`)
    process.exit(1)
  }
  const expected = `${JSON.stringify(messages[locale], null, 2)}\n`
  if (readFileSync(target, 'utf-8') !== expected) {
    console.error(`✗ chart-messages.${locale}.json 与 App locales 不同步：运行 node scripts/build-chart-messages.js`)
    process.exit(1)
  }
}
if (existsSync(join(WEB_DIR, 'shared/i18n/chart-messages.json'))) {
  console.error('✗ 旧单文件 chart-messages.json 仍在：运行 node scripts/build-chart-messages.js 清理')
  process.exit(1)
}
console.log(`✓ chart messages in sync (${LOCALES.length} locales)`);
