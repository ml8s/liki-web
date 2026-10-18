#!/usr/bin/env node
// check-i18n.js — Validate i18n consistency across locales.
// Checks: key alignment, price format consistency, untranslated fallbacks.

import fs from 'node:fs';
import path from 'node:path';

const I18N_DIR = path.resolve(import.meta.dirname, '..', 'i18n');
const APP_I18N_DIR = path.resolve(import.meta.dirname, '..', 'app', 'src', 'locales');
import SHARED_LOCALES from '../shared/locales.json' with { type: 'json' };
const errors = [];

function loadJSON(filepath) {
  try {
    return JSON.parse(fs.readFileSync(filepath, 'utf-8'));
  } catch (e) {
    errors.push(`${path.basename(filepath)}: invalid JSON — ${e.message}`);
    return null;
  }
}

function checkKeyAlignment(locales) {
  const names = Object.keys(locales);
  for (let i = 0; i < names.length; i++) {
    for (let j = i + 1; j < names.length; j++) {
      const a = names[i], b = names[j];
      const keysA = new Set(Object.keys(locales[a]));
      const keysB = new Set(Object.keys(locales[b]));

      for (const k of keysA) {
        if (!keysB.has(k)) errors.push(`${b}.json: missing key "${k}" (present in ${a}.json)`);
      }
      for (const k of keysB) {
        if (!keysA.has(k)) errors.push(`${a}.json: missing key "${k}" (present in ${b}.json)`);
      }
    }
  }
}

function getPath(obj, segments) {
  let cur = obj;
  for (const s of segments) {
    if (cur == null || typeof cur !== 'object') return undefined;
    cur = cur[s];
  }
  return cur;
}

// Cross-surface terminology anchors: the marketing site and the SPA maintain
// separate i18n catalogs (static build vs runtime), so user-visible terms can
// drift silently. Anchors below MUST resolve to the same value in every locale.
const TERM_ANCHORS = [
  { name: 'partner program (nav)', marketing: 'footer.partners', app: 'nav.partner' },
  { name: 'partner program (page title)', marketing: 'footer.partners', app: 'router.partnerCenter' },
  { name: 'brand slogan', marketing: 'site.slogan', app: 'auth.slogan' },
  // 标签标题后缀（「<页面名> · <站点名>」）在两个 surface 上必须是同一个名字。
  { name: 'site name (tab title)', marketing: 'site.name', app: 'site.name' },
  { name: 'service: bond', marketing: 'index.services.bond.title', app: 'services.bond.label' },
];

function checkCrossSurfaceTerms(marketingLocales) {
  for (const locale of SHARED_LOCALES.locales) {
    const marketing = marketingLocales[locale];
    const app = loadJSON(path.join(APP_I18N_DIR, `${locale}.json`));
    if (!marketing || !app) {
      errors.push(`cross-surface: missing locale file for "${locale}" (marketing=${Boolean(marketing)} app=${Boolean(app)})`);
      continue;
    }
    for (const anchor of TERM_ANCHORS) {
      const m = marketing[anchor.marketing];
      const a = getPath(app, anchor.app.split('.'));
      if (m === undefined || a === undefined) {
        errors.push(`cross-surface [${locale}] ${anchor.name}: missing key (marketing ${anchor.marketing}=${JSON.stringify(m)}, app ${anchor.app}=${JSON.stringify(a)})`);
      } else if (m !== a) {
        errors.push(`cross-surface [${locale}] ${anchor.name}: marketing "${anchor.marketing}"=${JSON.stringify(m)} ≠ app "${anchor.app}"=${JSON.stringify(a)}`);
      }
    }
  }
}

function checkPrices(locales) {
  // Check that price-format keys (containing "price" or "unlock" or "ctaDesc")
  // use consistent currency markers. All should use USD ($) now.
  const priceKeyRe = /price|unlock|ctaDesc/i;
  const currencyRe = /([¥£€]|HK\$|CNY|HKD|USD)/g;

  for (const [locale, data] of Object.entries(locales)) {
    for (const [key, value] of Object.entries(data)) {
      if (typeof value !== 'string') continue;
      if (!priceKeyRe.test(key)) continue;

      const matches = value.match(currencyRe);
      if (matches) {
        for (const m of matches) {
          if (m !== '$' || (m === '$' && value.includes('HK$'))) {
            // Allow $ only if it's not HK$
            if (m === '$' && !value.includes('HK$') && !value.includes('HKD')) continue;
          }
          errors.push(`${locale}.json: "${key}" contains non-USD currency "${m}" — use "$" only`);
        }
      }
    }
  }
}

// Marketing sources that own the marketing catalog (web/i18n/*.json). The app
// catalog (web/app/src/locales/*.json) lives on the SPA surface and is validated
// by web/app/src/tests/i18n.test.ts — two catalogs, two surfaces, two checkers.
const MARKETING_SRC = path.resolve(import.meta.dirname, '..', 'src');
const MARKETING_EXTENSIONS = new Set(['.astro', '.ts']);

// Keys are referenced as t(locale, 'key'), t(locale, `template`) or as a
// key-holding attribute (labelKey="key"). i18next.t() and data-i18n attributes
// belonged to the retired vanilla runtime; they are not part of the contract.
const KEY_ATTRIBUTE_RE =
  /(?:labelKey|titleKey|subtitleKey|questionKey|answerKey|benefitsTitleKey|exampleTitleKey|stepsTitleKey|questionsTitleKey|faqTitleKey|ctaKey|intentKey|descriptionKey)="([^"]+)"/g;
const LITERAL_KEY_RE = /t\(locale,\s*'([^']+)'\)/g;
const TEMPLATE_KEY_RE = /t\(locale,\s*`([^`]+)`\)/g;

function walkMarketingFiles(dir = MARKETING_SRC) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...walkMarketingFiles(full));
    else if (MARKETING_EXTENSIONS.has(path.extname(entry.name))) files.push(full);
  }
  return files;
}

// Every key the marketing sources reference, paired with the file that
// references it so failures point at the source rather than the catalog.
function collectMarketingKeys() {
  const root = path.resolve(import.meta.dirname, '..');
  const literal = new Map();
  const template = new Map();
  for (const filepath of walkMarketingFiles()) {
    const file = path.relative(root, filepath);
    const content = fs.readFileSync(filepath, 'utf-8');
    for (const re of [LITERAL_KEY_RE, KEY_ATTRIBUTE_RE]) {
      for (const m of content.matchAll(re)) literal.set(m[1], file);
    }
    for (const m of content.matchAll(TEMPLATE_KEY_RE)) template.set(m[1], file);
  }
  return { literal, template };
}

function checkHtmlKeys(locales) {
  const allKeys = new Set();
  for (const data of Object.values(locales)) {
    for (const k of Object.keys(data)) allKeys.add(k);
  }

  const { literal, template } = collectMarketingKeys();

  for (const [key, file] of literal) {
    if (!allKeys.has(key)) errors.push(`${file}: t(locale, '${key}') not found in any locale file`);
  }
  for (const [tmpl, file] of template) {
    for (const key of expandTemplateKey(tmpl)) {
      if (!allKeys.has(key)) {
        errors.push(`${file}: t(locale, \`${tmpl}\`) → "${key}" not found in any locale file`);
      }
    }
  }
}

// Page names used by BaseLayout's dynamic keys: t(locale, `${page}.meta.ogTitle`).
// The source of truth is the pages directory itself; never keep a second copy.
const PAGES_DIR = path.resolve(import.meta.dirname, '../src/pages/[locale]');
const KNOWN_PAGES = fs.readdirSync(PAGES_DIR)
  .filter(f => f.endsWith('.astro'))
  .map(f => (f === 'index.astro' ? 'index' : path.basename(f, '.astro')));

function expandTemplateKey(tmpl, fileContent) {
  const keys = [];

  // Pattern: ${ns}.field${n}suffix — e.g. naming.example keys iterating 1..36
  const namedNsMatch = tmpl.match(/\$\{ns\}\.(\w+)(\$\{n\})(\w)?/);
  if (namedNsMatch) {
    // Find const ns = '...' or ns = '...' in the same file
    const nsRe = /(?:const\s+)?ns\s*=\s*'([^']+)'/;
    const nsM = (fileContent || '').match(nsRe);
    const prefix = nsM ? nsM[1] : 'naming.example';
    const base = namedNsMatch[1];
    const suffix = namedNsMatch[3] || '';
    const maxN = prefix === 'naming.example' ? 36 : 10;
    for (let n = 1; n <= maxN; n++) {
      keys.push(`${prefix}.${base}${n}${suffix}`);
    }
    return keys;
  }

  // Pattern: ${page}.suffix — used in BaseLayout
  const pageMatch = tmpl.match(/^\$\{page\}\.(.+)$/);
  if (pageMatch) {
    const suffix = pageMatch[1];
    for (const page of KNOWN_PAGES) {
      keys.push(`${page}.${suffix}`);
    }
    return keys;
  }

  // Pattern: ${ns}.suffix
  const nsMatch = tmpl.match(/^\$\{ns\}\.(.+)$/);
  if (nsMatch) {
    const suffix = nsMatch[1];
    // Find all ns usages: ns={'...'} or ns={nsValue} or const ns = '...'
    const nsValues = new Set();
    const nsValsRe = /ns=\{'([^']+)'\}/g;
    let m;
    const fc = fileContent || '';
    while ((m = nsValsRe.exec(fc)) !== null) {
      nsValues.add(m[1]);
    }
    const nsConstRe = /(?:const\s+)?ns\s*=\s*'([^']+)'/g;
    while ((m = nsConstRe.exec(fc)) !== null) {
      nsValues.add(m[1]);
    }
    for (const val of nsValues) {
      keys.push(`${val}.${suffix}`);
    }
    return keys;
  }

  // Pattern: ${s} in .map() iteration — e.g. ['bazi','ziwei','character','advice'].map(s => ...)
  const sIterMatch = tmpl.match(/(\$\{s\})/);
  if (sIterMatch) {
    const prefix = tmpl.substring(0, tmpl.indexOf('${s}'));
    const suffix = tmpl.substring(tmpl.indexOf('${s}') + 4);
    // Find array map patterns: ['a','b','c'].map(s =>  or  [\"a\",\"b\"].map(s =>
    const arrayRe = /\['([^']+)'\]|\["([^"]+)"\]/g;
    const fc = fileContent || '';
    let arrM;
    while ((arrM = arrayRe.exec(fc)) !== null) {
      const items = (arrM[1] || arrM[2]).split(/',\s*'|",\s*"/);
      for (const item of items) {
        keys.push(prefix + item + suffix);
      }
      // Only use the first array match
      if (keys.length > 0) break;
    }
    return keys;
  }

  return [];
}

function addKnownCompoundKeys(used) {
  const META = ['metaDesc','meta.ogTitle','meta.ogDesc','meta.twitterTitle','meta.twitterDesc','title','heading'];
  for (const p of KNOWN_PAGES) for (const s of META) used.add(`${p}.${s}`);
  for (const key of [
    'brand', 'cta.primary', 'cta.secondary', 'cta.bottom', 'stepsTitle', 'faqTitle',
    'steps.s1', 'steps.s2', 'steps.s3', 'steps.s4',
  ]) used.add(`scenario.${key}`);
  for (const faq of ['paid', 'data', 'accuracy']) {
    used.add(`scenario.faq.${faq}.q`);
    used.add(`scenario.faq.${faq}.a`);
  }
  for (const scenario of ['naming', 'mingshu', 'hepan']) {
    for (const key of ['title', 'subtitle', 'benefitsTitle', 'exampleTitle', 'questionsTitle']) {
      used.add(`scenario.${scenario}.${key}`);
    }
    for (let n = 1; n <= 3; n++) used.add(`scenario.${scenario}.q${n}`);
  }
  for (const benefit of ['birth', 'elements', 'meaning', 'review']) {
    used.add(`scenario.naming.benefit.${benefit}.title`);
    used.add(`scenario.naming.benefit.${benefit}.body`);
  }
  for (const benefit of ['personality', 'career', 'relationship', 'timing']) {
    used.add(`scenario.mingshu.benefit.${benefit}.title`);
    used.add(`scenario.mingshu.benefit.${benefit}.body`);
  }
  for (const section of ['personality', 'career', 'love', 'health', 'fortune', 'milestones']) {
    used.add(`scenario.mingshu.static.${section}.title`);
    used.add(`scenario.mingshu.static.${section}.body`);
  }
  for (const benefit of ['chemistry', 'friction', 'communication', 'timing']) {
    used.add(`scenario.hepan.benefit.${benefit}.title`);
    used.add(`scenario.hepan.benefit.${benefit}.body`);
  }
  for (const section of ['overview', 'wuxing', 'dayMaster', 'branches', 'rating']) {
    used.add(`scenario.hepan.static.${section}.title`);
    used.add(`scenario.hepan.static.${section}.body`);
  }
  // Capability landing pages share one content shape (see ScenarioLanding.astro):
  // title/subtitle/benefitsTitle/exampleTitle/questionsTitle, q1..q3, a fixed
  // benefit set, and the focus/method/output static sections.
  for (const scenario of ['bazi', 'divination', 'fengshui', 'huangli']) {
    for (const key of ['title', 'subtitle', 'benefitsTitle', 'exampleTitle', 'questionsTitle']) {
      used.add(`scenario.${scenario}.${key}`);
    }
    for (let n = 1; n <= 3; n++) used.add(`scenario.${scenario}.q${n}`);
    for (const benefit of ['structure', 'insight', 'timing', 'action']) {
      used.add(`scenario.${scenario}.benefit.${benefit}.title`);
      used.add(`scenario.${scenario}.benefit.${benefit}.body`);
    }
    for (const section of ['focus', 'method', 'output']) {
      used.add(`scenario.${scenario}.static.${section}.title`);
      used.add(`scenario.${scenario}.static.${section}.body`);
    }
  }
  for (const f of ['name','gender','quote','source','footer']) {
    for (let n=1;n<=36;n++) used.add(`naming.example.${f}${n}`);
  }
  for (let n=1;n<=36;n++) for (const t of ['a','b','c','d']) used.add(`naming.example.tag${n}${t}`);
  const namingStatic = ['a11y.examples'];
  for (const k of namingStatic) used.add(`naming.${k}`);
}

function checkUnusedKeys(locales) {
  const { literal, template } = collectMarketingKeys();
  const used = new Set(literal.keys());
  for (const tmpl of template.keys()) {
    for (const key of expandTemplateKey(tmpl)) used.add(key);
  }

  // Pre-populate known compound keys that are constructed dynamically at build time
  addKnownCompoundKeys(used);

  const allKeys = new Set();
  for (const data of Object.values(locales)) {
    for (const k of Object.keys(data)) allKeys.add(k);
  }

  for (const key of allKeys) {
    if (!used.has(key)) {
      errors.push(`unused i18n key: "${key}" — defined but never referenced in source files`);
    }
  }
}

function checkEmptyValues(locales) {
  for (const [locale, data] of Object.entries(locales)) {
    for (const [key, value] of Object.entries(data)) {
      if (value === '' || value === null || value === undefined) {
        errors.push(`${locale}.json: "${key}" is empty`);
      }
    }
  }
}

function main() {
  const files = fs.readdirSync(I18N_DIR).filter(f => f.endsWith('.json'));
  if (files.length < 2) {
    console.log('Need at least 2 locale files to compare.');
    process.exit(0);
  }

  const locales = {};
  for (const file of files) {
    const name = file.replace('.json', '');
    const data = loadJSON(path.join(I18N_DIR, file));
    if (data) locales[name] = data;
  }

  checkKeyAlignment(locales);
  checkCrossSurfaceTerms(locales);
  checkPrices(locales);
  checkEmptyValues(locales);
  checkHtmlKeys(locales);
  checkUnusedKeys(locales);

  if (errors.length > 0) {
    console.error(`\n${errors.length} i18n violation(s):\n`);
    for (const e of errors) console.error(`  ✗ ${e}`);
    console.error();
    process.exit(1);
  }

  console.log(`✓ ${Object.keys(locales).length} locale(s) checked, no violations.`);
}

main();
