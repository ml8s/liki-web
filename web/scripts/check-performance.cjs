#!/usr/bin/env node
// check-performance.cjs — enforce the small set of mobile/loading contracts we
// can verify statically: viewport, non-blocking console modules, isolated
// console CSS, and a CSS budget for shared surfaces.

const fs = require('fs');
const path = require('path');

const web = path.resolve(__dirname, '..');
const errors = [];

function read(file, folder = '') {
  return fs.readFileSync(path.join(web, folder, file), 'utf8');
}

function checkFile(file) {
  const source = read(file);
  const rel = `web/${file}`;

  if (!/<meta\s+name="viewport"\s+content="[^"]*width=device-width[^"]*"/i.test(source)) {
    errors.push(`${rel}: missing mobile viewport`);
  }
  if (!/initial-scale=1/i.test(source.match(/<meta\s+name="viewport"[^>]*>/i)?.[0] || '')) {
    errors.push(`${rel}: viewport must include initial-scale=1`);
  }

  for (const match of source.matchAll(/<script\b[^>]*\bsrc=["']([^"']+)["'][^>]*>/gi)) {
    const tag = match[0];
    if (tag.startsWith('<script src=') && !/\b(?:defer|async)\b/i.test(tag)) {
      errors.push(`${rel}: external script must use defer/async: ${match[1]}`);
    }
  }

  if (/^\/(?:kol|scout)$/.test(`/${path.basename(file, '.html')}`) || file === 'admin.html') {
    if (source.includes('/css/app.css')) {
      errors.push(`${rel}: console must not load marketing app.css bundle`);
    }
    for (const css of ['tokens.css', 'console.css']) {
      if (!source.includes(`/css/${css}`)) {
        errors.push(`${rel}: missing required console stylesheet /css/${css}`);
      }
    }
  }
}

['kol.html', 'scout.html', 'admin.html'].forEach(checkFile);

const budgets = [
  ['tokens.css', 5 * 1024],
  ['console.css', 12 * 1024],
];
for (const [file, max] of budgets) {
  const size = fs.statSync(path.join(web, 'css', file)).size;
  if (size > max) {
    errors.push(`web/css/${file}: ${size} bytes exceeds ${max} byte budget`);
  }
}

const consoleCss = read('console.css', 'css');
if (!consoleCss.includes('@media (max-width: 640px)')) {
  errors.push('web/css/console.css: missing primary mobile breakpoint');
}
if (!consoleCss.includes('@media (hover: none) and (pointer: coarse)')) {
  errors.push('web/css/console.css: missing coarse pointer / mobile touch contract');
}

if (errors.length) {
  console.error(`✗ ${errors.length} performance/mobile issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log('✓ console mobile viewport, CSS isolation, budgets, and deferred modules checked');
