#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const web = path.resolve(import.meta.dirname, '..');
const canonical = fs.readFileSync(path.join(web, 'css/tokens.css'), 'utf8');
const app = fs.readFileSync(path.join(web, 'app/src/styles/app.css'), 'utf8');
const staticCss = fs.readFileSync(path.join(web, 'css/app.css'), 'utf8');
const appExpected = [
  '--brand-primary',
  '--brand-primary-hover',
  '--brand-primary-soft',
  '--brand-text',
  '--brand-line',
];
// Static CSS consumes semantic aliases; primitive values stay in tokens.css.
const staticExpected = ['--color-gold-light'];
let failed = false;

for (const token of appExpected) {
  if (!canonical.includes(token)) {
    console.error(`✗ canonical token missing: ${token}`);
    failed = true;
  }
}

for (const token of staticExpected) {
  if (!canonical.includes(token) || !staticCss.includes(token)) {
    console.error(`✗ static design token missing or unconsumed: ${token}`);
    failed = true;
  }
}

if (failed) process.exit(1);
console.log('✓ canonical design tokens are consistent');
