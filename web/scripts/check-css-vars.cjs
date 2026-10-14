#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const web = path.resolve(__dirname, '..');
const roots = ['css', 'src', path.join('app', 'src')];
const extensions = new Set(['.astro', '.css', '.vue']);
const files = [];

function walk(directory) {
  if (!fs.existsSync(directory)) return;
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (extensions.has(path.extname(entry.name))) files.push(full);
  }
}

roots.forEach(root => walk(path.join(web, root)));

const definitions = new Set();
const uses = [];
for (const file of files) {
  const source = fs.readFileSync(file, 'utf8');
  for (const match of source.matchAll(/(--[a-z0-9-]+)\s*:/gi)) definitions.add(match[1]);
  for (const match of source.matchAll(/var\(\s*(--[a-z0-9-]+)/gi)) {
    uses.push({
      name: match[1],
      file,
      line: source.slice(0, match.index).split('\n').length,
    });
  }
}

const unresolved = uses.filter(use => !definitions.has(use.name));
if (unresolved.length) {
  console.error(`✗ ${unresolved.length} unresolved CSS variable(s):`);
  for (const use of unresolved) {
    console.error(`  ✗ ${path.relative(web, use.file)}:${use.line} ${use.name}`);
  }
  process.exit(1);
}

console.log('✓ all consumed CSS custom properties resolve in source');
