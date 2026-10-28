#!/usr/bin/env node
// Product chat is BFF-only. This gate prevents retired direct-runtime routes,
// build switches, and callback endpoints from returning through source changes.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const DEFAULT_ROOT = path.resolve(import.meta.dirname, '..', '..');
const SELF_PATH = fileURLToPath(import.meta.url);
const SOURCE_ROOTS = ['cmd', 'internal', 'web/src', 'web/app/src', 'web/e2e', 'web/scripts'];
const EXTENSIONS = new Set(['.go', '.js', '.mjs', '.ts', '.vue']);
const RETIRED_MARKERS = [
  '/rt/',
  'VITE_CHAT_TRANSPORT',
  'LIKI_RUN_CALLBACK_SECRET',
  '/api/internal/runs/completed',
  '/api/internal/introspect',
];

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === 'test-results') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full, files);
    else if (EXTENSIONS.has(path.extname(entry.name))) files.push(full);
  }
  return files;
}

export function findRetiredChatTransports(root = DEFAULT_ROOT) {
  const findings = [];
  for (const relativeRoot of SOURCE_ROOTS) {
    const absoluteRoot = path.join(root, relativeRoot);
    if (!fs.existsSync(absoluteRoot)) continue;
    for (const file of walk(absoluteRoot)) {
      if (file === SELF_PATH) continue;
      const source = fs.readFileSync(file, 'utf-8');
      const relativeFile = path.relative(root, file);
      source.split('\n').forEach((line, index) => {
        for (const marker of RETIRED_MARKERS) {
          if (line.includes(marker)) {
            findings.push(`${relativeFile}:${index + 1}: retired chat transport marker ${JSON.stringify(marker)}`);
          }
        }
      });
    }
  }
  return findings;
}

function main() {
  const positional = process.argv.filter((arg, index) => index >= 2 && !arg.startsWith('--'));
  const findings = findRetiredChatTransports(path.resolve(positional[0] ?? DEFAULT_ROOT));
  if (findings.length) {
    console.error(`❌ retired chat transport markers found (${findings.length}):`);
    for (const finding of findings) console.error(`  ✗ ${finding}`);
    process.exit(1);
  }
  console.log('✓ product chat transport is BFF-only');
}

function isMainModule() {
  return process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);
}

if (isMainModule()) main();
