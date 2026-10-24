#!/usr/bin/env node
// check-classes.js — Validate CSS class names in templates against app.css.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const rootArg = process.argv[2];
const ROOT = rootArg ? path.resolve(rootArg) : path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

if (!fs.existsSync(path.join(ROOT, 'css/app.css')) || !fs.existsSync(path.join(ROOT, 'src'))) {
  console.error('usage: check-classes.js [project-root-with-css/app.css-and-src]');
  process.exit(1);
}

// ── Load CSS classes via simple line scanning ──
const css = ['app.css']
  .map(f => fs.readFileSync(path.join(ROOT, 'css', f), 'utf-8'))
  .join('\n');
const cssClasses = new Set();

for (const line of css.split('\n')) {
  const trimmed = line.trim();
  // Match lines that start with a CSS class definition.
  // e.g., .foo {  .foo:hover {  .min-w-\[130px\] {  .mt-0\.5 {
  const m = trimmed.match(/^\.([\w-]+(?:\\[\w.:[\]-]+)*)/);
  if (m) {
    // Normalize escape sequences: \: → :  \. → .  \[ → [  \] → ]
    let cls = m[1].replace(/\\:/g, ':').replace(/\\\./g, '.').replace(/\\\[/g, '[').replace(/\\\]/g, ']');
    // Strip trailing pseudo-class (e.g., :hover, :focus, :active)
    cls = cls.replace(/(?::(?:hover|focus|active|focus-visible|disabled|not-defined|first-child|last-child))$/, '');
    cssClasses.add(cls);
  }
}

// 契约：模板里写死的类名都必须在 app.css 里定义——包括模板化属性里的字符串字面量
// （class={`...`} / class={cond ? 'a' : 'b'}），其中反引号模板只算 `${...}` 之外的片段。
// 求不出值的不检查（变量、表达式结果）：那些不是「写死的类名」。没有豁免名单——需要
// 豁免就说明 app.css 缺定义，去补定义，而不是放宽检查。

// class={ ... } 的配对右括号位置（跳过字符串与模板字面量里的括号）。
// start 指向 `{` 之后的第一个字符，所以初始深度是 1。
function readBraced(source, start) {
  let depth = 1;
  for (let i = start; i < source.length; i++) {
    const ch = source[i];
    if (ch === '{') depth++;
    else if (ch === '}') {
      depth--;
      if (depth === 0) return { text: source.slice(start, i), end: i + 1 };
    } else if (ch === '"' || ch === "'" || ch === '`') {
      i = skipQuoted(source, i);
    }
  }
  return null;
}

function skipQuoted(source, start) {
  const quote = source[start];
  for (let i = start + 1; i < source.length; i++) {
    if (source[i] === '\\') { i++; continue; }
    if (source[i] === quote) return i;
  }
  return source.length - 1;
}

// 表达式里的字符串字面量；反引号模板去掉 `${...}` 插值，只留写死的片段。
// 比较运算右侧的字面量是数据（type === 'article'、l.code === locale），不是类名。
function stringLiterals(expression) {
  const literals = [];
  let cursor = 0;
  for (const m of expression.matchAll(/'([^']*)'|"([^"]*)"|`([^`]*)`/g)) {
    const before = expression.slice(cursor, m.index).trimEnd();
    cursor = m.index + m[0].length;
    if (/[=!]==?$/.test(before)) continue;
    const value = m[1] ?? m[2] ?? m[3] ?? '';
    literals.push(m[3] === undefined ? value : value.replace(/\$\{[^}]*\}/g, ' '));
  }
  return literals;
}

// 模板里所有能静态求值的 class 值（字面量属性 + 模板化属性里的字面量片段）。
function classLiterals(source) {
  const literals = [];
  const attrRe = /(?<![\w-])class=(?:"([^"]*)"|\{)/g;
  let m;
  while ((m = attrRe.exec(source)) !== null) {
    if (m[1] !== undefined) {
      literals.push(m[1]);
      continue;
    }
    const expression = readBraced(source, attrRe.lastIndex);
    if (!expression) continue;
    attrRe.lastIndex = expression.end;
    literals.push(...stringLiterals(expression.text));
  }
  return literals;
}

// ── Scan Astro templates ──
const srcDir = path.join(ROOT, 'src');
let errors = [];

function scanDir(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      scanDir(full);
    } else if (entry.name.endsWith('.astro')) {
      for (const literal of classLiterals(fs.readFileSync(full, 'utf-8'))) {
        for (const cls of literal.split(/\s+/)) {
          // 残留的表达式片段（拼接、三元、插值符号）不是写死的类名。
          if (!cls || /[{}()'"+$]/.test(cls)) continue;
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
