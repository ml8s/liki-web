import fs from 'fs';
import path from 'path';

const ROOT = path.resolve(__dirname, '..', '..', '..');

export function readJS(name) {
  return fs.readFileSync(`${ROOT}/web/js/${name}.js`, 'utf-8');
}

export function el(tag, props = {}) {
  const e = {
    _tag: tag,
    value: props.value || '',
    disabled: props.disabled || false,
    textContent: props.text || '',
    className: props.className || '',
    checked: props.checked || false,
    classList: {
      _cs: [],
      add(c) { this._cs.push(c); },
      remove(c) { this._cs = this._cs.filter(x => x !== c); },
      contains(c) { return this._cs.includes(c); },
    },
    addEventListener(type, fn) { this['_ls' + type] = fn; },
    trim() { return String(this.value).trim(); },
  };
  return e;
}
