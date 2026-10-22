#!/usr/bin/env node
// check-error-codes.js — 错误码契约检查器（单源强制）：
//   1) 从 Go 源码抽取全部错误码（RespondError 字面量 / AppError{Code:} / ErrCode 常量）；
//   2) 校验 App 三语 locale 的 errors.code.* 全部有文案（规则：每个码都要文案，无豁免——
//      内部码也给中性文案，若将来泄漏到用户路径仍是本地化文案）；
//   3) 校验生成物 docs/error-codes.md 与抽取结果一致（新鲜度）。
// 与 chart-messages 同模式：约定 → 检查器 → 生成物三件套。
import fs from 'node:fs';
import path from 'node:path';

const WEB = path.resolve(import.meta.dirname, '..');
const ROOT = path.resolve(WEB, '..');
const DOC = path.join(ROOT, 'docs', 'error-codes.md');
const LOCALES = ['zh-Hans', 'zh-Hant', 'en'];
const errors = [];

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full, files);
    else if (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')) files.push(full);
  }
  return files;
}

// 与运行时契约一致：RespondError(w, status, "code", ...) / AppError{Code: "code"} /
// httputil 的 ErrCode 常量定义。
export function collectGoCodes() {
  const codes = new Map();
  for (const file of walk(path.join(ROOT, 'internal'))) {
    const src = fs.readFileSync(file, 'utf-8');
    const rel = path.relative(ROOT, file);
    for (const m of src.matchAll(/RespondError\([^,]+,\s*[^,]+,\s*"([a-z_]+)"/g)) {
      if (!codes.has(m[1])) codes.set(m[1], rel);
    }
    for (const m of src.matchAll(/AppError\{[^}]*Code:\s*"([a-z_]+)"/g)) {
      if (!codes.has(m[1])) codes.set(m[1], rel);
    }
    for (const m of src.matchAll(/ErrCode\w+\s*=\s*"([a-z_]+)"/g)) {
      if (!codes.has(m[1])) codes.set(m[1], rel);
    }
  }
  return codes;
}

function appCodes(locale) {
  const data = JSON.parse(fs.readFileSync(path.join(WEB, 'app/src/locales', `${locale}.json`), 'utf-8'));
  return new Set(Object.keys(data.errors?.code ?? {}));
}

function renderDoc(codes) {
  const lines = [
    '# Error Codes',
    '',
    '<!-- 本文件由 web/scripts/check-error-codes.js 生成：请勿手改。改动错误码后重跑 `node web/scripts/check-error-codes.js --update`。 -->',
    '',
    '每个 Go 错误码都必须在 App 三语 `errors.code.*` 中有文案（检查器 `make check-web-static` 强制）。',
    '状态类端点只读且响应不含 PII；`payment-status` 能力 token 与 intent id 属 capability-URL 凭证（见 ARCHITECTURE.md）。',
    '',
    '| Code | 定义处 |',
    '| --- | --- |',
    ...[...codes.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([code, file]) => `| \`${code}\` | \`${file}\` |`),
    '',
  ];
  return lines.join('\n');
}

const codes = collectGoCodes();
for (const locale of LOCALES) {
  const known = appCodes(locale);
  for (const code of codes.keys()) {
    if (!known.has(code)) errors.push(`${locale}.json: missing errors.code.${code} (defined at ${codes.get(code)})`);
  }
}

if (process.argv.includes('--update')) {
  fs.mkdirSync(path.dirname(DOC), { recursive: true });
  fs.writeFileSync(DOC, renderDoc(codes));
  console.log(`✓ docs/error-codes.md written (${codes.size} codes)`);
  process.exit(0);
}

if (errors.length) {
  console.error(`\n${errors.length} error-code contract violation(s):\n`);
  for (const e of errors) console.error(`  ✗ ${e}`);
  console.error('\n规则：每个 Go 错误码都要有 App 三语文案；补文案后重跑 --update 刷新 docs/error-codes.md。\n');
  process.exit(1);
}

if (!fs.existsSync(DOC) || fs.readFileSync(DOC, 'utf-8') !== renderDoc(codes)) {
  console.error('❌ docs/error-codes.md 过期：运行 node web/scripts/check-error-codes.js --update 刷新。');
  process.exit(1);
}
console.log(`✓ error codes in sync (${codes.size} codes × ${LOCALES.length} locales + docs freshness)`);
