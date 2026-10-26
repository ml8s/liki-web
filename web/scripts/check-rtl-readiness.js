#!/usr/bin/env node
// Consultation/Me are logical-property-first. This gate prevents physical
// direction declarations from returning before an RTL locale is introduced.
import fs from 'node:fs';
import path from 'node:path';

const app = path.join(path.resolve(import.meta.dirname, '..', '..'), 'web', 'app');
const targets = [
  path.join(app, 'src/components/flow'),
  path.join(app, 'src/views/MeView.vue'),
  path.join(app, 'src/views/DomainFlowView.vue'),
];
const violations = [];

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
      continue;
    }
    if (!/\.(vue|css)$/.test(entry.name)) continue;
    const source = fs.readFileSync(full, 'utf8');
    const relative = path.relative(app, full).replaceAll('\\', '/');
    for (const [index, line] of source.split(/\r?\n/).entries()) {
      const physical = line.match(/(?:^|[;{\n]\s*)(margin-left|margin-right|padding-left|padding-right|left|right)\s*:/);
      const alignment = line.match(/(?:^|[;{\n]\s*)text-align\s*:\s*(left|right)\b/);
      if (physical) violations.push(`${relative}:${index + 1} physical property "${physical[1]}"`);
      if (alignment) violations.push(`${relative}:${index + 1} physical text alignment "${alignment[1]}"`);
    }
  }
}

for (const target of targets) {
  if (fs.existsSync(target)) {
    if (fs.statSync(target).isDirectory()) {
      walk(target);
      continue;
    }
    const source = fs.readFileSync(target, 'utf8');
    const relative = path.relative(app, target).replaceAll('\\', '/');
    for (const [index, line] of source.split(/\r?\n/).entries()) {
      const physical = line.match(/(?:^|[;{\n]\s*)(margin-left|margin-right|padding-left|padding-right|left|right)\s*:/);
      const alignment = line.match(/(?:^|[;{\n]\s*)text-align\s*:\s*(left|right)\b/);
      if (physical) violations.push(`${relative}:${index + 1} physical property "${physical[1]}"`);
      if (alignment) violations.push(`${relative}:${index + 1} physical text alignment "${alignment[1]}"`);
    }
  } else {
    violations.push(`missing RTL readiness target: ${target}`);
  }
}

if (violations.length) {
  console.error(`✗ ${violations.length} RTL readiness issue(s):`);
  for (const violation of violations) console.error(`  ✗ ${violation}`);
  process.exit(1);
}

console.log('✓ Consultation/Me styles use logical direction properties');
