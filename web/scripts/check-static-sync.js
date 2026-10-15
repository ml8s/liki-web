#!/usr/bin/env node
// check-static-sync.js — 校验 web/css 与 web/public/css 双份物理副本一致。
// 背景：web/css 是源（Astro build 期读取），public/css 是部署资产（Astro build 拷入
// dist，Go dev 直接读 public/）。两份必须逐字节一致——漂移 = 部署内容与源不符。
import fs from 'node:fs';
import path from 'node:path';

const WEB = path.resolve(import.meta.dirname, '..');
const SOURCE = path.join(WEB, 'css');
const PUBLISHED = path.join(WEB, 'public', 'css');

function walk(dir) {
  const entries = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) entries.push(...walk(full));
    else entries.push(path.relative(dir, full));
  }
  return entries;
}

for (const dir of [SOURCE, PUBLISHED]) {
  if (!fs.existsSync(dir)) {
    console.error(`✗ 缺目录：${path.relative(WEB, dir)}`);
    process.exit(1);
  }
}

const source = walk(SOURCE).sort();
const published = walk(PUBLISHED).sort();
let fail = 0;

for (const file of [...new Set([...source, ...published])]) {
  const inSource = source.includes(file);
  const inPublished = published.includes(file);
  if (inSource !== inPublished) {
    console.error(`✗ css/${file} 存在于一侧但另一侧缺失`);
    fail = 1;
  } else if (!fs.readFileSync(path.join(SOURCE, file)).equals(fs.readFileSync(path.join(PUBLISHED, file)))) {
    console.error(`✗ css/${file} 内容不一致`);
    fail = 1;
  }
}

if (fail === 0) console.log(`✓ css/ ↔ public/css/ 一致（${source.length} 个文件）`);
process.exit(fail);
