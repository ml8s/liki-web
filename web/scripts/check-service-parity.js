#!/usr/bin/env node
// check-service-parity.js — Marketing CTA parameters must be understood by the app.
//
// The marketing site hardcodes scenario=/service= CTA values; the SPA owns the
// runtime vocabulary (scenario whitelist + service keys). They are related but
// distinct facts, so we assert a subset invariant instead of merging them:
// every value marketing emits MUST be recognized by the app.
import fs from 'node:fs';
import path from 'node:path';

const WEB = path.resolve(import.meta.dirname, '..');
const errors = [];

function walk(dir, exts, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!['node_modules', 'dist'].includes(entry.name)) walk(full, exts, files);
    } else if (exts.some(e => entry.name.endsWith(e))) files.push(full);
  }
  return files;
}

// ── Marketing: collect scenario="x" / service="x" literals ──
const marketing = new Set();
for (const file of walk(path.join(WEB, 'src'), ['.astro'])) {
  const text = fs.readFileSync(file, 'utf8');
  for (const m of text.matchAll(/\b(?:scenario|service)="([a-z]+)"/g)) marketing.add(m[1]);
}

// ── App: scenario whitelist (composables/scenario.ts) ──
const scenarioSrc = fs.readFileSync(path.join(WEB, 'app/src/composables/scenario.ts'), 'utf8');
const appScenarios = new Set([...scenarioSrc.matchAll(/scenario === '([a-z]+)'/g)].map(m => m[1]));

// ── App: service keys (config/chat.ts) ──
const chatSrc = fs.readFileSync(path.join(WEB, 'app/src/config/chat.ts'), 'utf8');
const appServices = new Set([...chatSrc.matchAll(/(?:key|id): '([a-z]+)'/g)].map(m => m[1]));

for (const value of [...marketing].sort()) {
  const known = appScenarios.has(value) || appServices.has(value);
  if (!known) {
    errors.push(
      `marketing CTA emits "${value}" but the app knows neither a scenario (${[...appScenarios].join(', ')}) nor a service (${[...appServices].join(', ')})`,
    );
  }
}

if (errors.length) {
  console.error(`\n${errors.length} service-parity violation(s):\n`);
  for (const e of errors) console.error(`  ✗ ${e}`);
  process.exit(1);
}
console.log(`✓ ${marketing.size} marketing CTA value(s) all recognized by the app`);
