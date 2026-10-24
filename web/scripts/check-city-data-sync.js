#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const source = path.resolve(
  process.env.LIKI_CITY_DATA ?? path.join(process.cwd(), '..', 'liki', 'engine', 'internal', 'engine', 'city', 'cities_data.json'),
)
const target = path.resolve(process.cwd(), 'web', 'public', 'data', 'cities_data.json')

if (!fs.existsSync(source)) {
  console.log('! city data source not found; skipping cross-repo sync check')
  process.exit(0)
}
if (!fs.existsSync(target)) {
  console.error(`✗ missing published city data: ${target}`)
  process.exit(1)
}
if (!fs.readFileSync(source).equals(fs.readFileSync(target))) {
  console.error('✗ city data is stale; copy engine/internal/engine/city/cities_data.json to web/public/data/cities_data.json')
  process.exit(1)
}
console.log('✓ city data matches the engine source')
