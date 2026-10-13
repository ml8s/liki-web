#!/usr/bin/env node
// check-css-utility-freeze.cjs — Freeze the hand-written utility-class surface.
//
// web/css historically accumulates Tailwind-named but hand-written utilities
// (no JIT, incomplete). New code must use semantic classes or brand tokens.
// The baseline below is the frozen allow-list: additions fail, removals are
// encouraged (update the baseline when you delete one).
const fs = require('fs');
const path = require('path');

const CSS_DIR = path.resolve(__dirname, '..', 'css');
const BASELINE = path.resolve(__dirname, 'css-utility-baseline.json');

const UTILITY_RE = /^(flex|grid|block|inline|hidden|relative|absolute|sticky|fixed|text-|bg-|border|rounded|font-|max-w-|min-w-|w-|h-|gap-|items-|justify-|space-|overflow|opacity|z-|shadow|transition|duration|whitespace|truncate|mx-|my-|mt-|mb-|ml-|mr-|px-|py-|pt-|pb-|pl-|pr-|p-|m-|leading|tracking|object|list|select|pointer|align|self|col|row-|order|sr-only|antialiased|underline|italic|uppercase|lowercase|capitalize)/;

const FILES = ['app.css'];

function utilities(file) {
  const classes = new Set();
  for (const line of fs.readFileSync(path.join(CSS_DIR, file), 'utf8').split('\n')) {
    const m = line.trim().match(/^\.([\w-]+(?:\\[\w.:\[\]-]+)*)/);
    if (m) {
      const cls = m[1].replace(/\\([:.])/g, '$1').replace(/\\/g, '');
      if (UTILITY_RE.test(cls)) classes.add(cls);
    }
  }
  return classes;
}

const baseline = JSON.parse(fs.readFileSync(BASELINE, 'utf8'));
const errors = [];
for (const file of FILES) {
  const current = utilities(file);
  const frozen = new Set(baseline[file] ?? []);
  for (const cls of current) {
    if (!frozen.has(cls)) errors.push(`${file}: new utility class ".${cls}" — use semantic classes / brand tokens instead`);
  }
  const removed = [...frozen].filter(c => !current.has(c));
  if (removed.length) {
    console.log(`ℹ ${file}: ${removed.length} frozen utilit${removed.length === 1 ? 'y' : 'ies'} removed (${removed.join(', ')}) — shrink the baseline in css-utility-baseline.json`);
  }
}

if (errors.length) {
  console.error(`\n${errors.length} CSS utility-freeze violation(s):\n`);
  for (const e of errors) console.error(`  ✗ ${e}`);
  process.exit(1);
}
console.log('✓ CSS utility surface matches the frozen baseline');
