#!/usr/bin/env node
// check-static-sync.cjs — 校验 web/{js,css} 与 web/public/{js,css} 双份物理副本一致。
// 背景：web/js|css 是源（Astro build 期 readFileSync + Go dev 兜底），public/js|css 是
// 部署资产（Astro build 拷入 dist）。两份必须逐字节一致——漂移 = 部署内容与源不符。
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const web = path.resolve(__dirname, '..');
let fail = 0;
for (const dir of ['js', 'css']) {
  const a = path.join(web, dir), b = path.join(web, 'public', dir);
  if (!fs.existsSync(a) || !fs.existsSync(b)) {
    console.error(`✗ 缺目录：${dir} 或 public/${dir}`);
    process.exit(1);
  }
  try {
    execSync(`diff -r "${a}" "${b}"`, { stdio: 'pipe' });
    console.log(`✓ ${dir}/ ↔ public/${dir}/ 一致`);
  } catch (e) {
    console.error(`✗ ${dir}/ 与 public/${dir}/ 漂移（改了源必须同步另一份）：\n${e.stdout}`);
    fail = 1;
  }
}
process.exit(fail);
