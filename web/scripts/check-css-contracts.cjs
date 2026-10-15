#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const web = path.resolve(__dirname, '..');
const errors = [];

function read(file) {
  return fs.readFileSync(path.join(web, file), 'utf8');
}

function cssSource(file) {
  return read(file).replace(/\/\*[\s\S]*?\*\//g, '');
}

const staticCss = cssSource(path.join('css', 'app.css'));
const primitiveUses = staticCss.match(/var\(--brand-[a-z0-9-]+\)/g);
if (primitiveUses?.length) {
  errors.push(`css/app.css consumes ${primitiveUses.length} primitive token(s); use a semantic alias`);
}

const rawColor = staticCss.match(/#[0-9a-fA-F]{3,8}\b|rgba?\(/g);
if (rawColor?.length) {
  errors.push(`css/app.css has ${rawColor.length} raw color value(s); consume a token from css/tokens.css`);
}

const cssFiles = ['css/app.css'];
const vueRoot = path.join(web, 'app', 'src');
function walk(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.name.endsWith('.vue')) cssFiles.push(path.relative(web, full));
  }
}
walk(vueRoot);

for (const file of cssFiles) {
  const source = cssSource(file);
  for (const match of source.matchAll(/z-index\s*:\s*\d+/gi)) {
    errors.push(`${file}: raw z-index "${match[0]}"`);
  }
}

const allowedWidths = new Set(['480', '640', '720', '768', '1023.98', '1024']);
for (const file of cssFiles) {
  const source = cssSource(file);
  for (const match of source.matchAll(/@media[^{]*?\(\s*(?:max|min)-width\s*:\s*([\d.]+)px/g)) {
    if (!allowedWidths.has(match[1])) errors.push(`${file}: nonstandard breakpoint ${match[1]}px`);
  }
}

if (errors.length) {
  console.error(`✗ ${errors.length} CSS contract issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log('✓ static colors, z-index, and breakpoint contracts checked');
