#!/usr/bin/env node
// check-classes.cjs — Validate CSS class names in templates against app.css.
const fs = require('fs');
const path = require('path');
const ROOT = path.resolve(__dirname, '..');

// ── Load CSS classes via simple line scanning ──
const css = ['app.css']
  .map(f => fs.readFileSync(path.join(ROOT, 'css', f), 'utf-8'))
  .join('\n');
const cssClasses = new Set();

for (const line of css.split('\n')) {
  const trimmed = line.trim();
  // Match lines that start with a CSS class definition.
  // e.g., .foo {  .foo:hover {  .min-w-\[130px\] {  .mt-0\.5 {
  const m = trimmed.match(/^\.([\w-]+(?:\\[\w.:\[\]-]+)*)/);
  if (m) {
    // Normalize escape sequences: \: → :  \. → .  \[ → [  \] → ]
    let cls = m[1].replace(/\\:/g, ':').replace(/\\\./g, '.').replace(/\\\[/g, '[').replace(/\\\]/g, ']');
    // Strip trailing pseudo-class (e.g., :hover, :focus, :active)
    cls = cls.replace(/(?::(?:hover|focus|active|focus-visible|disabled|not-defined|first-child|last-child))$/, '');
    cssClasses.add(cls);
  }
}

// ── Dynamic classes (applied by JS/Vue, not literal in template) ──
const dynamic = new Set([
  'msg-row', 'msg-row-user', 'msg-row-advisor', 'has-error',
  'login-overlay', 'order-card', 'error-toast', 'expired-banner',
  'brand', 'skip-link',
  'msg-asst-wrap', 'msg-asst', 'msg-user', 'msg-time',
  'chat-shell', 'chat-header', 'chat-input-bar',
  'btn-send', 'btn-stop', 'btn-chat-circle', 'btn-newchat',
  'btn-header', 'lang-switcher', 'liki-header', 'liki-footer',
  'liki-page', 'phase-status',
  'name-preview-card', 'name-preview-detail', 'name-preview-title',
  'name-preview-gender', 'name-preview-quote',
  'skill-icon',
  'select-overlay', 'login-card', 'select-card',
  'lb-hero', 'lb-form-section',
  'hero-brand', 'cta-section', 'page-main', 'page-card',
  'qr-modal', 'qr-modal-close',
  'py-2.5', // defined as .py-2\.5 in CSS (backslash-escaped dot)
  'relative', // Tailwind utility, used in chat.astro for lang-switcher
]);

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
          if (dynamic.has(cls)) continue;
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
