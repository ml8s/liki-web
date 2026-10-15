#!/usr/bin/env node
// check-classes.js — Validate CSS class names in templates against app.css.
import fs from 'node:fs';
import path from 'node:path';
const ROOT = path.resolve(import.meta.dirname, '..');

// ── Load CSS classes via simple line scanning ──
const css = ['app.css']
  .map(f => fs.readFileSync(path.join(ROOT, 'css', f), 'utf-8'))
  .join('\n');
const cssClasses = new Set();

for (const line of css.split('\n')) {
  const trimmed = line.trim();
  // Match lines that start with a CSS class definition.
  // e.g., .foo {  .foo:hover {  .min-w-\[130px\] {  .mt-0\.5 {
  const m = trimmed.match(/^\.([\w-]+(?:\\[\w.:[\]-]+)*)/);
  if (m) {
    // Normalize escape sequences: \: → :  \. → .  \[ → [  \] → ]
    let cls = m[1].replace(/\\:/g, ':').replace(/\\\./g, '.').replace(/\\\[/g, '[').replace(/\\\]/g, ']');
    // Strip trailing pseudo-class (e.g., :hover, :focus, :active)
    cls = cls.replace(/(?::(?:hover|focus|active|focus-visible|disabled|not-defined|first-child|last-child))$/, '');
    cssClasses.add(cls);
  }
}

// 契约：字面 class="..." 里的每个类名都必须在 app.css 里定义；含 `{`/`}`/`+`/引号
// 的模板化 class 属性不参与检查（无法静态求值）。这里没有豁免名单——需要豁免就说明
// app.css 缺定义，去补定义，而不是放宽检查。

// ── Scan Astro templates ──
const srcDir = path.join(ROOT, 'src');
let errors = [];

function scanDir(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      scanDir(full);
    } else if (entry.name.endsWith('.astro')) {
      const html = fs.readFileSync(full, 'utf-8');
      const attrRe = /class="([^"]*?)"/g;
      let m;
      while ((m = attrRe.exec(html)) !== null) {
        const val = m[1];
        if (/[{{+'"}]/.test(val)) continue;
        for (const cls of val.split(/\s+/)) {
          if (!cls) continue;
          if (!cssClasses.has(cls)) {
            errors.push(`${path.relative(srcDir, full)}: .${cls}`);
          }
        }
      }
    }
  }
}
scanDir(srcDir);

// ── Report ──
if (errors.length > 0) {
  const unique = [...new Set(errors)];
  console.error(`❌ ${unique.length} 个未定义类名:\n`);
  for (const e of unique) {
    console.error(`   ${e}`);
  }
  process.exit(1);
} else {
  console.log('✓ 所有 Astro 模板中的类名都有对应 CSS 定义');
}
