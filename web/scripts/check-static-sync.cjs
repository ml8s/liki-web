#!/usr/bin/env node
// check-static-sync.cjs — 校验 web/{js,css} 与 web/public/{js,css} 双份物理副本一致。
// 背景：web/js|css 是源（Astro build 期 readFileSync + Go dev 兜底），public/js|css 是
// 部署资产（Astro build 拷入 dist）。两份必须逐字节一致——漂移 = 部署内容与源不符。
const fs = require('fs');
const path = require('path');
const web = path.resolve(__dirname, '..');
let fail = 0;

function walk(dir, base) {
  const entries = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    const rel = path.relative(base, full);
    if (entry.isDirectory()) entries.push(...walk(full, base));
    else entries.push(rel);
  }
  return entries;
}

function filesEqual(a, b) {
  const bufA = fs.readFileSync(a);
  const bufB = fs.readFileSync(b);
  return bufA.equals(bufB);
}

for (const dir of ['js', 'css']) {
  const a = path.join(web, dir), b = path.join(web, 'public', dir);
  if (!fs.existsSync(a) && !fs.existsSync(b)) {
    if (dir === 'js') {
      console.log(`✓ js/ retired; public/js absent`);
      continue;
    }
    console.error(`✗ 缺目录：${dir} 或 public/${dir}`);
    fail = 1;
    continue;
  }
  const filesA = walk(a, a).sort();
  const filesB = walk(b, b).sort();
  const allFiles = [...new Set([...filesA, ...filesB])];
  let drifted = false;
  for (const f of allFiles) {
    const inA = filesA.includes(f), inB = filesB.includes(f);
    if (inA !== inB) {
      console.error(`✗ ${dir}/${f} 存在于一侧但另一侧缺失`);
      drifted = true;
    } else if (!filesEqual(path.join(a, f), path.join(b, f))) {
      console.error(`✗ ${dir}/${f} 内容不一致`);
      drifted = true;
    }
  }
  if (!drifted) console.log(`✓ ${dir}/ ↔ public/${dir}/ 一致`);
  else fail = 1;
}
process.exit(fail);
