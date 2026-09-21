#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const web = path.resolve(__dirname, '..');
const canonical = fs.readFileSync(path.join(web, 'css/tokens.css'), 'utf8');
const app = fs.readFileSync(path.join(web, 'app/src/styles/app.css'), 'utf8');
const consoleCss = fs.readFileSync(path.join(web, 'css/console.css'), 'utf8');
const expected = [
  '--brand-primary',
  '--brand-primary-hover',
  '--brand-primary-soft',
  '--brand-gold',
  '--brand-text',
  '--brand-line',
];
let failed = false;

for (const token of expected) {
  if (!canonical.includes(token)) {
    console.error(`✗ canonical token missing: ${token}`);
    failed = true;
  }
}

for (const token of expected) {
  if (!app.includes(token)) {
    console.error(`✗ app does not consume canonical token: ${token}`);
    failed = true;
  }
}

for (const banned of ['#5f4bc2', '#4f3db0']) {
  if (consoleCss.toLowerCase().includes(banned)) {
    console.error(`✗ console.css hardcodes brand color: ${banned}`);
    failed = true;
  }
}

if (failed) process.exit(1);
console.log('✓ canonical design tokens are consistent');
