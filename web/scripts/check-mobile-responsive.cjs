#!/usr/bin/env node
// check-mobile-responsive.cjs — enforce the shared mobile ergonomics contract.

const fs = require('fs');
const path = require('path');

const web = path.resolve(__dirname, '..');
const errors = [];
const targets = [
  'app/src/views/AppLayout.vue',
  'app/src/components/app/AppTabbar.vue',
  'app/src/views/OrdersView.vue',
  'app/src/components/chat/SessionList.vue',
];

function read(rel) {
  return fs.readFileSync(path.join(web, rel), 'utf8');
}

for (const rel of targets) {
  const source = read(rel);
  if (/(height:\s*100vh)(?!.*100dvh)/s.test(source)) {
    errors.push(`${rel}: uses 100vh without a 100dvh fallback`);
  }
  if (/(padding-bottom:(?:\s*)(?:24px|16px|88px))/.test(source) && !source.includes('env(safe-area-inset-bottom)')) {
    errors.push(`${rel}: bottom padding lacks safe-area handling`);
  }
}

const tabbar = read('app/src/components/app/AppTabbar.vue');
if (!tabbar.includes('env(safe-area-inset-bottom)')) {
  errors.push('AppTabbar: bottom navigation must respect iOS safe area');
}
if (!/height:\s*4[4-9]px|height:\s*5[0-9]px/.test(tabbar)) {
  errors.push('AppTabbar: mobile navigation links should be at least 44px tall');
}

const composable = read('app/src/composables/useMobile.ts');
if (!composable.includes('onBeforeUnmount')) {
  errors.push('useMobile: media listener must be removed on unmount');
}

if (errors.length) {
  console.error(`✗ ${errors.length} mobile responsive issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}

console.log('✓ shared app mobile layout, safe areas, and touch ergonomics checked');
