#!/usr/bin/env node
// check-form-kit.js — own raw field controls and shrink migration debt.
//
// New raw Element Plus field tags/imports outside the shared form kit fail.
// Baseline files are known legacy debt; when one no longer contains raw field
// controls, the script requires its baseline entry to be removed.
import fs from 'node:fs';
import path from 'node:path';

const web = path.resolve(import.meta.dirname, '..');
const roots = ['src', 'shared', path.join('app', 'src')];
const extensions = new Set(['.vue']);

const controlNames = new Set([
  'ElAutocomplete', 'ElCascader', 'ElCheckbox', 'ElDatePicker', 'ElInput',
  'ElInputNumber', 'ElOption', 'ElRadioButton', 'ElRadioGroup', 'ElSelect',
  'ElTimePicker',
]);
const controlTags = [...controlNames].map(name => name
  .replace(/^El/, '')
  .replace(/([a-z])([A-Z])/g, '$1-$2')
  .toLowerCase());

const baseline = new Set([
  'app/src/components/birth-profiles/BirthForm.vue',
  'app/src/components/birth-profiles/ChartPreview.vue',
  'app/src/components/chat/BirthProfilePicker.vue',
  'app/src/components/chat/ChatComposer.vue',
  'app/src/components/partner/PartnerPayoutDialog.vue',
  'app/src/components/partner/PartnerReferralCard.vue',
  'app/src/components/partner/PartnerRoleStatus.vue',
  'app/src/views/AccountView.vue',
  'app/src/views/FengshuiView.vue',
  'app/src/views/HepanView.vue',
  'app/src/views/LoginView.vue',
  'app/src/views/admin/AdminAdministratorsView.vue',
  'app/src/views/admin/AdminEntitlementsView.vue',
  'app/src/views/admin/AdminFinanceView.vue',
  'app/src/views/admin/AdminOrdersView.vue',
  'app/src/views/admin/AdminPartnersView.vue',
  'app/src/views/admin/AdminPayoutsView.vue',
  'app/src/views/admin/AdminSubscriptionsView.vue',
  'shared/components/fengshui/FengshuiLayoutFields.vue',
]);

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

const tagPattern = new RegExp(`<el-(${controlTags.join('|')})(?=[\\s/>])`, 'gi');
const importPattern = /import\s*\{([^}]+)\}\s*from\s*['"]element-plus['"]/g;
const errors = [];
const usedBaseline = new Set();

for (const file of files) {
  const relative = path.relative(web, file).replaceAll(path.sep, '/');
  if (relative.startsWith('shared/components/form/')) continue;

  const source = fs.readFileSync(file, 'utf8');
  const matches = [
    ...[...source.matchAll(tagPattern)].map(match => `raw <el-${match[1]}>`),
    ...[...source.matchAll(importPattern)].flatMap(match => match[1]
      .split(',')
      .map(specifier => specifier.trim().replaceAll(/^type\s+/.u, ''))
      .filter(specifier => controlNames.has(specifier))
      .map(specifier => `direct ${specifier} import`)),
  ];

  if (!matches.length) continue;
  if (!baseline.has(relative)) {
    errors.push(`${relative}: ${matches.join(', ')} — use the shared form kit`);
  } else {
    usedBaseline.add(relative);
  }
}

const staleBaseline = [...baseline].filter(entry => !usedBaseline.has(entry));
if (staleBaseline.length) {
  errors.push(...staleBaseline.map(entry => `${entry}: obsolete migration baseline entry`));
}

if (errors.length) {
  console.error(`✗ ${errors.length} form-kit boundary issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log(`✓ raw field controls remain within migration baseline (${baseline.size} file(s))`);
