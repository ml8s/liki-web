#!/usr/bin/env node
// build-sitemap.cjs — 生成 sitemap.xml，lastmod 基于 git 最后一次修改时间
const fs = require('fs');
const { execSync } = require('child_process');
const path = require('path');

const PROJECT = path.resolve(__dirname, '../..');
const OUT = path.resolve(PROJECT, 'web/sitemap.xml');

const LOCALES = ['zh-Hant', 'zh-Hans', 'en'];
const DEFAULT_LOCALE = 'zh-Hant';

// Structure (which pages exist) is derived from the pages directory — the only
// copy. Per-page sitemap metadata is configuration: defaults plus overrides,
// so a new page is included automatically without editing this file.
const PAGES_DIR = path.resolve(__dirname, '../src/pages/[locale]');
const DEFAULT_PAGE_META = { freq: 'monthly', priority: '0.5' };
const PAGE_META = {
  '':          { freq: 'weekly', priority: '1.0' },
  skills:      { freq: 'monthly', priority: '0.6' },
  naming:      { freq: 'weekly', priority: '0.8' },
  mingshu:     { freq: 'weekly', priority: '0.8' },
  hepan:       { freq: 'weekly', priority: '0.8' },
  contact:     { freq: 'monthly', priority: '0.4' },
  disclaimer:  { freq: 'monthly', priority: '0.3' },
  privacy:     { freq: 'yearly', priority: '0.2' },
  terms:       { freq: 'yearly', priority: '0.2' },
};

function discoverPages() {
  return fs.readdirSync(PAGES_DIR)
    .filter(f => f.endsWith('.astro'))
    .map(f => path.basename(f, '.astro'))
    .sort((a, b) => (a === 'index' ? -1 : b === 'index' ? 1 : a.localeCompare(b)))
    .map(name => ({
      path: name === 'index' ? '' : name,
      src: `web/src/pages/[locale]/${name}.astro`,
      ...(PAGE_META[name === 'index' ? '' : name] ?? DEFAULT_PAGE_META),
    }));
}

const PAGES = discoverPages();

const SPECIAL = [
  { loc: 'https://liki.hk/llms.txt',        src: 'web/llms.txt',           freq: 'weekly',  priority: '0.7' },
  { loc: 'https://liki.hk/skills/SKILL.md',  src: 'web/skills/liki/SKILL.md',      freq: 'monthly', priority: '0.5' },
];

function gitLastmod(file) {
  try {
    const full = path.resolve(PROJECT, file);
    const dir = path.dirname(full);
    const rel = path.relative(dir, full);
    const lastmod = execSync(`git -C "${dir}" log -1 --format="%cI" -- "${rel}"`, { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'ignore'] }).trim();
    return lastmod || new Date().toISOString().split('T')[0];
  } catch {
    return new Date().toISOString().split('T')[0];
  }
}

function url(loc, lastmod, freq, priority, links) {
  let out = `  <url>\n    <loc>${loc}</loc>\n`;
  for (const l of links) {
    out += `    <xhtml:link rel="alternate" hreflang="${l.hreflang}" href="${l.href}"/>\n`;
  }
  out += `    <lastmod>${lastmod}</lastmod>\n    <changefreq>${freq}</changefreq>\n    <priority>${priority}</priority>\n  </url>`;
  return out;
}

const lines = [];
for (const page of PAGES) {
  const lastmod = gitLastmod(page.src);
  const links = LOCALES.map(l => ({
    hreflang: l,
    href: `https://liki.hk/${l}${page.path ? '/' + page.path : ''}/`,
  }));
  links.push({ hreflang: 'x-default', href: `https://liki.hk/${DEFAULT_LOCALE}${page.path ? '/' + page.path : ''}/` });
  for (const l of LOCALES) {
    lines.push(url(`https://liki.hk/${l}${page.path ? '/' + page.path : ''}/`, lastmod, page.freq, page.priority, links));
  }
}

for (const s of SPECIAL) {
  const lastmod = gitLastmod(s.src);
  const links = [{ hreflang: 'x-default', href: s.loc }];
  lines.push(url(s.loc, lastmod, s.freq, s.priority, links));
}

const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
  xmlns:xhtml="http://www.w3.org/1999/xhtml">
${lines.join('\n')}
</urlset>
`;

fs.writeFileSync(OUT, xml, 'utf-8');
console.log(`✓ sitemap.xml — ${PAGES.length+SPECIAL.length} URLs`);
