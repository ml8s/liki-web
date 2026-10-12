#!/usr/bin/env node
// check-i18n.cjs — Validate i18n consistency across locales.
// Checks: key alignment, price format consistency, untranslated fallbacks.

const fs = require('fs');
const path = require('path');

const I18N_DIR = path.resolve(__dirname, '..', 'i18n');
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

function checkHtmlKeys(locales) {
  const allKeys = new Set();
  for (const [name, data] of Object.entries(locales)) {
    for (const k of Object.keys(data)) allKeys.add(k);
  }

  const astroFiles = gatherAstroFiles();
  const tRe = /t\(locale,\s*'([^']+)'\)/g;

  for (const { file, filepath } of astroFiles) {
    const content = fs.readFileSync(filepath, 'utf-8');
    let m;
    while ((m = tRe.exec(content)) !== null) {
      if (!allKeys.has(m[1])) {
        errors.push(`${file}: t(locale, '${m[1]}') not found in any locale file`);
      }
    }
    // Check template literal keys
    const tmplRe = /t\(locale,\s*`([^`]+)`\)/g;
    while ((m = tmplRe.exec(content)) !== null) {
      for (const key of expandTemplateKey(m[1])) {
        if (!allKeys.has(key)) {
          errors.push(`${file}: t(locale, \`${m[1]}\`) → "${key}" not found in any locale file`);
        }
      }
    }
  }
}

function gatherAstroFiles() {
  const webDir = path.resolve(__dirname, '..');
  const pagesDir = path.join(webDir, 'src', 'pages');
  const componentsDir = path.join(webDir, 'src', 'components');
  const layoutsDir = path.join(webDir, 'src', 'layouts');
  const files = [];
  for (const dir of [pagesDir, componentsDir, layoutsDir]) {
    walkDir(dir, files);
  }
  return files;
}

function walkDir(dir, files) {
  if (!fs.existsSync(dir)) return;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) { walkDir(full, files); }
    else if (entry.name.endsWith('.astro')) {
      files.push({ file: path.relative(path.resolve(__dirname, '..'), full), filepath: full });
    }
  }
}

// Known page names used in BaseLayout: t(locale, `${page}.meta.ogTitle`)
const KNOWN_PAGES = ['about','contact','disclaimer','privacy','terms','index','skills','partners','hepan','mingshu','naming'];

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
    'brand', 'cta.primary', 'cta.secondary', 'cta.bottom', 'stepsTitle', 'faqTitle', 'demoNote',
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
  for (const f of ['name','gender','quote','source','footer']) {
    for (let n=1;n<=36;n++) used.add(`naming.example.${f}${n}`);
  }
  for (let n=1;n<=36;n++) for (const t of ['a','b','c','d']) used.add(`naming.example.tag${n}${t}`);
  const namingStatic = ['a11y.examples'];
  for (const k of namingStatic) used.add(`naming.${k}`);
  for (const s of ['hero.title','hero.subtitle','cta.title']) {
    used.add(`hepan.${s}`);
    used.add(`mingshu.${s}`);
  }
  for (const k of ['hero.title','hero.subtitle','hero.desc','features','install.title','install.method1','install.method2']) {
    used.add(`index.${k}`);
  }
  for (const skill of ['mingli','qiming','wengua','fengshui']) {
    for (const k of ['title','desc']) used.add(`index.skills.${skill}.${k}`);
  }
}

function checkUnusedKeys(locales) {
  const webDir = path.resolve(__dirname, '..');
  const used = new Set();

  const keyPatterns = [
    /t\(locale,\s*'([^']+)'\)/g,
    /t\(locale,\s*"([^"]+)"\)/g,
    /i18next\.t\('([^']+)'\)/g,
    /i18next\.t\("([^"]+)"\)/g,
    /\bT\('([^']+)'\)/g,
    /\bT\("([^"]+)"\)/g,
    /\bt\('([^']+)'\)/g,
    /\bt\("([^"]+)"\)/g,
    /data-i18n(?:-\w+)?="([^"]+)"/g,
    /data-i18n(?:-\w+)?='([^']+)'/g,
    /(?:labelKey|titleKey|subtitleKey|questionKey|answerKey|benefitsTitleKey|exampleTitleKey|stepsTitleKey|questionsTitleKey|faqTitleKey|ctaKey|intentKey|descriptionKey)="([^"]+)"/g,
  ];

  function collectKeys(content, filepath) {
    for (const re of keyPatterns) {
      let m;
      while ((m = re.exec(content)) !== null) {
        used.add(m[1]);
      }
    }
    // Template literal keys: t(locale, `...${n}...`)
    const tmplRe = /t\(locale,\s*`([^`]+)`\)/g;
    let tm;
    while ((tm = tmplRe.exec(content)) !== null) {
      for (const key of expandTemplateKey(tm[1])) {
        used.add(key);
      }
    }
  }

  function scanDir(dir) {
    if (!fs.existsSync(dir)) return;
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        if (!['node_modules', '.git', 'dist', 'i18n'].includes(entry.name)) {
          scanDir(full);
        }
      } else if (/\.(astro|js|html|cjs|mjs)$/.test(entry.name)) {
        const content = fs.readFileSync(full, 'utf-8');
        collectKeys(content, full);
      }
    }
  }

  scanDir(webDir);

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
