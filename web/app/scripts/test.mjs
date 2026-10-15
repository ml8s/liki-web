import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const vitestCli = fileURLToPath(new URL('../node_modules/vitest/vitest.mjs', import.meta.url))

const result = spawnSync(process.execPath, [vitestCli, 'run', ...process.argv.slice(2)], {
  stdio: 'inherit',
})

process.exit(result.status ?? 1)
