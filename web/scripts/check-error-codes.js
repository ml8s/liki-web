#!/usr/bin/env node
// check-error-codes.js — 错误码契约检查器（双向单源强制）：
//   1) 从 Go 源码抽取全部错误码（RespondError 字面量 / AppError{Code:} / ErrCode 常量）；
//   2) App 三语 locale 与 Go 错误码必须双向一致：不能缺文案，也不能保留孤儿文案；
//   3) 校验生成物 docs/error-codes.md 与抽取结果一致（新鲜度）。
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const DEFAULT_ROOT = path.resolve(import.meta.dirname, '..', '..');
const LOCALES = ['zh-Hans', 'zh-Hant', 'en'];

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
export function collectGoCodes(root = DEFAULT_ROOT) {
  const codes = new Map();
  for (const file of walk(path.join(root, 'internal'))) {
    const src = fs.readFileSync(file, 'utf-8');
    const rel = path.relative(root, file);
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

function localePath(root, locale) {
  return path.join(root, 'web/app/src/locales', `${locale}.json`);
}

function appCodes(root, locale) {
  const source = fs.readFileSync(localePath(root, locale), 'utf-8');
  if (!source.trim()) return new Set();
  return new Set(Object.keys(JSON.parse(source).errors?.code ?? {}));
}

function frontendOnlyCodes(root) {
  const inventory = path.join(root, 'web/scripts/error-codes-frontend.json');
  if (!fs.existsSync(inventory)) return new Set();
  return new Set(JSON.parse(fs.readFileSync(inventory, 'utf-8')).codes ?? []);
}

export function contractErrors(root = DEFAULT_ROOT) {
  const codes = collectGoCodes(root);
  const knownCodes = new Set([...codes.keys(), ...frontendOnlyCodes(root)]);
  const errors = [];
  for (const locale of LOCALES) {
    const known = appCodes(root, locale);
    for (const code of knownCodes) {
      if (!known.has(code)) {
        const source = codes.has(code) ? `defined at ${codes.get(code)}` : 'frontend-only definition';
        errors.push(`${locale}.json: missing errors.code.${code} (${source})`);
      }
    }
    for (const code of known) {
      if (!knownCodes.has(code)) errors.push(`${locale}.json: orphan errors.code.${code} (no Go or frontend definition)`);
    }
  }
  return errors;
}

export function renderDoc(codes) {
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

function docPath(root) {
  return path.join(root, 'docs', 'error-codes.md');
}

function main() {
  const positional = process.argv.filter((arg, index) => index >= 2 && !arg.startsWith('--'));
  const root = path.resolve(positional[0] ?? DEFAULT_ROOT);
  const doc = docPath(root);
  const codes = collectGoCodes(root);
  const errors = contractErrors(root);
  if (errors.length) {
    console.error(`\n${errors.length} error-code contract violation(s):\n`);
    for (const error of errors) console.error(`  ✗ ${error}`);
    console.error('\n规则：Go 与 App 三语错误码必须双向一致；补/删文案后重跑 --update 刷新 docs/error-codes.md。\n');
    process.exit(1);
  }

  if (process.argv.includes('--update')) {
    fs.mkdirSync(path.dirname(doc), { recursive: true });
    fs.writeFileSync(doc, renderDoc(codes));
    console.log(`✓ docs/error-codes.md written (${codes.size} codes)`);
    return;
  }

  if (!fs.existsSync(doc) || fs.readFileSync(doc, 'utf-8') !== renderDoc(codes)) {
    console.error('❌ docs/error-codes.md 过期：运行 node web/scripts/check-error-codes.js --update 刷新。');
    process.exit(1);
  }
  console.log(`✓ error codes in sync (${codes.size} codes × ${LOCALES.length} locales + docs freshness)`);
}

function isMainModule() {
  return process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);
}

if (isMainModule()) main();
