import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..', '..');
const directories = ['web/app/src/components', 'web/app/src/views'];
let failed = false;

function walk(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      walk(full);
      continue;
    }
    if (/\.(vue|css)$/.test(entry.name)) check(full);
  }
}

function check(file) {
  const source = fs.readFileSync(file, 'utf8');
  const matches = source.match(/#[0-9a-fA-F]{3,8}\b|rgba?\(/g) || [];
  if (matches.length) {
    console.error(`✗ raw color in ${path.relative(root, file)}: ${matches.length}`);
    failed = true;
  }
}

directories.forEach(directory => walk(path.join(root, directory)));
if (failed) process.exit(1);
console.log('✓ component colors use semantic tokens');
