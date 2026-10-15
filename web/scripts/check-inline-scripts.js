#!/usr/bin/env node
// Inventory CSP-relevant script tags and reject growth beyond the reviewed
// baseline. Migration should shrink this baseline; it should never be raised
// to admit a new executable inline script.

import fs from 'node:fs';
import path from 'node:path';

const WEB_DIR = path.resolve(import.meta.dirname, '..');
const BASELINE = path.join(import.meta.dirname, 'inline-scripts-baseline.json');
const ROOTS = [path.join(WEB_DIR, 'src'), path.join(WEB_DIR, 'app', 'index.html')];
const distIndex = process.argv.indexOf('--dist');
const DIST_DIR = distIndex === -1 ? null : path.resolve(WEB_DIR, process.argv[distIndex + 1]);
const EXECUTABLE_TYPES = new Set(['', 'module', 'text/javascript', 'application/javascript']);
const DATA_TYPES = new Set(['application/ld+json', 'application/json']);

function walk(dir, files = []) {
  if (!fs.statSync(dir).isDirectory()) {
    files.push(dir);
    return files;
  }
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const child = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(child, files);
    else if (/\.(astro|html)$/.test(entry.name)) files.push(child);
  }
  return files;
}

function classify(file, attrs) {
  const type = /type\s*=\s*["']([^"']+)["']/i.exec(attrs)?.[1]?.toLowerCase() || '';
  const src = /src\s*=\s*["'`]?([^"'\s>]+)/i.exec(attrs)?.[1] || '';
  const isAstro = file.endsWith('.astro');
  const isInline = /(?:^|\s)is:inline(?:\s|=|$)/i.test(attrs);

  if (src) return { kind: 'external', src, sri: /integrity\s*=/i.test(attrs) };
  if (DATA_TYPES.has(type)) return { kind: 'data-block' };
  if (!EXECUTABLE_TYPES.has(type)) return { kind: 'unknown-type', type };
  if (!isAstro || isInline) return { kind: 'executable-inline' };
  return { kind: 'astro-processed' };
}

function inventory() {
  const files = ROOTS.flatMap((root) => {
    if (!fs.existsSync(root)) return [];
    return fs.statSync(root).isDirectory()
      ? walk(root)
      : [root];
  }).sort();

  const findings = [];
  for (const file of files) {
    const relative = path.relative(WEB_DIR, file).replaceAll('\\', '/');
    const lines = fs.readFileSync(file, 'utf-8').split(/\r?\n/);
    lines.forEach((line, index) => {
      for (const match of line.matchAll(/<script\b([^>]*)>/gi)) {
        const item = classify(file, match[1]);
        const src = item.src || '';
        findings.push({
          file: relative,
          line: index + 1,
          kind: item.kind,
          src,
          thirdParty: /^https?:\/\//i.test(src),
          sri: Boolean(item.sri),
        });
      }
    });
  }
  return findings;
}

function summarize(findings) {
  const baseline = {};
  for (const finding of findings) {
    for (const category of [finding.kind, finding.thirdParty ? 'third-party' : null]) {
      if (!category) continue;
      const key = `${finding.file}:${category}`;
      baseline[key] = (baseline[key] || 0) + 1;
    }
    if (finding.kind === 'external' && !finding.sri) {
      const key = `${finding.file}:external-without-sri`;
      baseline[key] = (baseline[key] || 0) + 1;
    }
  }
  return Object.fromEntries(Object.entries(baseline).sort(([a], [b]) => a.localeCompare(b)));
}

const findings = inventory();
const actual = summarize(findings);

if (DIST_DIR) {
  const builtFiles = walk(DIST_DIR).filter((file) => /\.html$/i.test(file));
  const builtExecutable = [];
  const builtStyleAttributes = [];
  for (const file of builtFiles) {
    const relative = path.relative(WEB_DIR, file).replaceAll('\\', '/');
    const text = fs.readFileSync(file, 'utf-8');
    for (const match of text.matchAll(/<script\b([^>]*)>/gi)) {
      const attrs = match[1];
      const hasSrc = /\bsrc\s*=/i.test(attrs);
      const type = /\btype\s*=\s*["']([^"']+)["']/i.exec(attrs)?.[1]?.toLowerCase() || '';
      const isData = type === 'application/ld+json' || type === 'application/json';
      if (!hasSrc && !isData) builtExecutable.push(relative);
      if (/\sstyle\s*=/i.test(attrs)) builtStyleAttributes.push(`${relative}: style attribute`);
    }
  }
  if (builtExecutable.length) {
    console.error(`❌ built HTML contains ${builtExecutable.length} executable inline scripts:`);
    console.error(builtExecutable.map((file) => `  ${file}`).join('\n'));
    process.exit(1);
  }
  const styleElements = builtFiles
    .filter((file) => /<style\b/i.test(fs.readFileSync(file, 'utf-8')))
    .map((file) => path.relative(WEB_DIR, file).replaceAll('\\', '/'));
  if (styleElements.length) {
    console.error(`❌ built HTML contains ${styleElements.length} style elements:`);
    console.error(styleElements.map((file) => `  ${file}`).join('\n'));
    process.exit(1);
  }
  console.log(`✓ built HTML has no executable inline scripts (${builtFiles.length} pages)`);
  if (builtStyleAttributes.length) {
    console.log(
      `ℹ built HTML has ${builtStyleAttributes.length} style attributes; ` +
      "'unsafe-inline' remains an accepted Element Plus runtime exception"
    );
  }
  process.exit(0);
}

const executableInline = findings.filter((finding) => finding.kind === 'executable-inline');
if (executableInline.length) {
  console.error(`❌ source contains ${executableInline.length} executable inline scripts:`);
  console.error(executableInline.map((finding) => `  ${finding.file}:${finding.line}`).join('\n'));
  process.exit(1);
}

if (process.argv.includes('--update')) {
  fs.writeFileSync(BASELINE, JSON.stringify(actual, null, 2) + '\n');
  console.log(`✓ inline script baseline updated (${findings.length} findings)`);
  process.exit(0);
}

if (!fs.existsSync(BASELINE)) {
  console.error(`❌ missing baseline: ${path.relative(WEB_DIR, BASELINE)}`);
  process.exit(1);
}

const expected = JSON.parse(fs.readFileSync(BASELINE, 'utf-8'));
const regressions = [];
for (const [key, count] of Object.entries(actual)) {
  if (count > (expected[key] || 0)) regressions.push(`${key}: ${expected[key] || 0} -> ${count}`);
}
for (const key of Object.keys(expected)) {
  if (!(key in actual)) regressions.push(`${key}: disappeared from inventory; update baseline`);
}

for (const finding of findings.filter((item) => item.kind === 'executable-inline')) {
  console.log(`inline: ${finding.file}:${finding.line}`);
}

if (regressions.length) {
  console.error('❌ CSP-relevant script inventory grew:', `\n  ${regressions.join('\n  ')}`);
  process.exit(1);
}

console.log(`✓ inline script inventory matches baseline (${findings.length} findings)`);
