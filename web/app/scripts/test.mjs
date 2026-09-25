import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const vitestCli = fileURLToPath(new URL('../node_modules/vitest/vitest.mjs', import.meta.url))

// `npm run test` already runs once. Some callers still append `-- --run`,
// so normalize that redundant flag instead of forwarding `--run --run`.
const args = process.argv.slice(2)
const normalizedArgs = []
let seenRun = false

for (const arg of args) {
  if (arg === '--run') {
    if (seenRun) continue
    seenRun = true
  }
  normalizedArgs.push(arg)
}

const result = spawnSync(process.execPath, [vitestCli, 'run', ...normalizedArgs], {
  stdio: 'inherit',
})

process.exit(result.status ?? 1)
