#!/usr/bin/env node
// check-performance.js — guard the remaining source-level performance/loading
// contracts that are cheap to enforce before Vite builds the SPA.

import fs from 'node:fs';
import path from 'node:path';

const web = path.resolve(import.meta.dirname, '..');
const errors = [];

function read(file, folder = '') {
  return fs.readFileSync(path.join(web, folder, file), 'utf8');
}

const appHtml = read('index.html', 'app');
if (!/<meta\s+name="viewport"\s+content="[^"]*width=device-width[^"]*"/i.test(appHtml)) {
  errors.push('web/app/index.html: missing mobile viewport');
}
if (!/initial-scale=1/i.test(appHtml.match(/<meta\s+name="viewport"[^>]*>/i)?.[0] || '')) {
  errors.push('web/app/index.html: viewport must include initial-scale=1');
}
if (!/<meta\s+name="robots"\s+content="noindex"/i.test(appHtml)) {
  errors.push('web/app/index.html: authenticated shell must be noindex');
}

const budgets = [
  ['css/app.css', 40 * 1024],
  ['app/src/styles/app.css', 20 * 1024],
];
for (const [file, max] of budgets) {
  const size = fs.statSync(path.join(web, file)).size;
  if (size > max) errors.push(`web/${file}: ${size} bytes exceeds ${max} byte budget`);
}

if (errors.length) {
  console.error(`✗ ${errors.length} performance/mobile issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log('✓ app shell viewport/noindex and CSS budgets checked');
