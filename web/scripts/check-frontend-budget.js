#!/usr/bin/env node
// Built-output budget for the authenticated app shell. Raw limits catch bundle
// explosions locally; compressed limits approximate what mobile users download.
import fs from 'node:fs';
import path from 'node:path';

const dist = path.resolve(import.meta.dirname, '..', 'dist', 'app');
const errors = [];

function assetSize(file) {
  return fs.statSync(path.join(dist, file.replace(/^\/app\//, ''))).size;
}
function compressedSize(file, extension) {
  const compressed = path.join(dist, 'assets', `${path.basename(file)}.${extension}`);
  return fs.existsSync(compressed) ? fs.statSync(compressed).size : null;
}

const htmlPath = path.join(dist, 'index.html');
if (!fs.existsSync(htmlPath)) {
  console.error('✗ web/dist/app/index.html is missing; run the app build first');
  process.exit(1);
}

const html = fs.readFileSync(htmlPath, 'utf8');
const scripts = [...html.matchAll(/<script[^>]+src="([^"]+\.js)"/g)].map(match => match[1]);
const preloads = [...html.matchAll(/<link[^>]+rel="modulepreload"[^>]+href="([^"]+\.js)"/g)].map(match => match[1]);
const initialScripts = [...new Set([...scripts, ...preloads])];
const styles = [...html.matchAll(/<link[^>]+href="([^"]+\.css)"/g)].map(match => match[1]);

const budgets = [
  ['initial JavaScript', initialScripts, 320 * 1024, 120 * 1024],
  ['initial CSS', styles, 760 * 1024, 48 * 1024],
];

for (const [label, files, rawBudget, brotliBudget] of budgets) {
  if (!files.length) errors.push(`no ${label} assets found in app index.html`);
  const rawTotal = files.reduce((total, file) => total + assetSize(file), 0);
  if (rawTotal > rawBudget) {
    errors.push(`initial ${label}: ${(rawTotal / 1024).toFixed(1)} KiB exceeds ${(rawBudget / 1024).toFixed(0)} KiB`);
  }

  const brotliSizes = files
    .map(file => compressedSize(file, 'br'))
    .filter(size => typeof size === 'number');
  if (brotliSizes.length !== files.length) {
    errors.push(`missing brotli assets for initial ${label}`);
    continue;
  }
  const brotliTotal = brotliSizes.reduce((total, size) => total + size, 0);
  if (brotliTotal > brotliBudget) {
    errors.push(`initial ${label} brotli: ${(brotliTotal / 1024).toFixed(1)} KiB exceeds ${(brotliBudget / 1024).toFixed(0)} KiB`);
  }
}

if (errors.length) {
  console.error(`✗ ${errors.length} frontend budget issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log(`✓ frontend budget checked (${initialScripts.length} JS, ${styles.length} CSS initial assets)`);
