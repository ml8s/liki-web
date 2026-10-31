#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..');
const readJson = value => JSON.parse(fs.readFileSync(value, 'utf8'));
const intents = readJson(path.join(root, 'shared/config/intents.json'));
const marketing = readJson(path.join(root, 'routes/marketing.contract.json'));
const app = readJson(path.join(root, 'routes/app.contract.json'));
const expectedLocales = ['zh-Hans', 'zh-Hant', 'en'];
const errors = [];

if (intents.version !== 1) errors.push('intents.version must be 1');
if (marketing.version !== 1) errors.push('marketing.version must be 1');
if (app.version !== 1) errors.push('app.version must be 1');
if (JSON.stringify(marketing.locales) !== JSON.stringify(expectedLocales)) {
  errors.push('marketing.locales must equal zh-Hans, zh-Hant, en');
}

const intentIds = new Set();
const slugs = new Set();
const serviceParams = new Set();
const appPaths = new Set();
for (const intent of intents.intents ?? []) {
  for (const key of ['id', 'marketingSlug', 'serviceParam']) {
    if (typeof intent[key] !== 'string' || intent[key] === '') errors.push(`intent.${key} is required`);
  }
  if (intentIds.has(intent.id)) errors.push(`duplicate intent id: ${intent.id}`);
  if (slugs.has(intent.marketingSlug)) errors.push(`duplicate marketing slug: ${intent.marketingSlug}`);
  if (serviceParams.has(intent.serviceParam)) errors.push(`duplicate service param: ${intent.serviceParam}`);
  if (intent.appPath != null) {
    if (!intent.appPath.startsWith('/app/')) errors.push(`appPath must start with /app/: ${intent.appPath}`);
    if (appPaths.has(intent.appPath)) errors.push(`duplicate appPath: ${intent.appPath}`);
    appPaths.add(intent.appPath);
  }
  intentIds.add(intent.id);
  slugs.add(intent.marketingSlug);
  serviceParams.add(intent.serviceParam);
}

const appRoutes = app.routes ?? [];
const appRouteCounts = new Map();
for (const route of appRoutes) {
  if (typeof route !== 'string' || !route.startsWith('/app')) errors.push(`invalid app route: ${route}`);
  appRouteCounts.set(route, (appRouteCounts.get(route) ?? 0) + 1);
}
for (const [route, count] of appRouteCounts) {
  if (count > 1) errors.push(`duplicate app route: ${route}`);
}

const expectedSuffixes = [
  '',
  '/login',
  '/birth-profiles',
  '/birth-profiles/new',
  '/birth-profiles/:id',
  '/hepan',
  '/divination',
  '/fengshui',
  '/huangli',
  '/chat',
  '/chat/:sessionId',
  '/orders',
  '/partners',
  '/partners/kol',
  '/partners/scout',
  '/account',
  '/admin',
];
for (const suffix of expectedSuffixes) {
  if (appRouteCounts.get(`/app${suffix}`) !== 1) {
    errors.push(`app route missing or duplicated: /app${suffix}`);
  }
}
for (const retiredRoute of ['/app/consultation', '/app/me', '/app/classic']) {
  if (appRouteCounts.has(retiredRoute)) {
    errors.push(`retired route must not be canonical: ${retiredRoute}`);
  }
}

if (app.rollout?.previewDefault !== false || app.rollout?.status !== 'retired') {
  errors.push('app rollout must mark the consultation template retired');
}

const expectedCompatibility = [
  { from: '/app/consultation', to: '/app', preserve: ['query', 'hash'] },
  { from: '/app/me', to: '/app/account', preserve: ['query', 'hash'] },
  { from: '/app/classic', to: '/app', preserve: ['query', 'hash'] },
  { from: '/app/classic/:path', to: '/app/:path', preserve: ['query', 'hash'] },
];
if (JSON.stringify(app.experienceRedirects ?? []) !== JSON.stringify(expectedCompatibility)) {
  errors.push('retired template compatibility redirects are missing or stale');
}

const redirects = marketing.bareRedirects ?? [];
const retired = marketing.retired ?? [];
for (const route of redirects) {
  if (!intentIds.has(route.slug)) errors.push(`redirect references unknown intent: ${route.slug}`);
  if (route.status !== 302) errors.push(`redirect must use 302: ${route.path}`);
}
for (const route of retired) {
  if (route.status !== 404) errors.push(`retired route must use 404: ${route.path}`);
  if (slugs.has(route.path.slice(1))) errors.push(`retired route conflicts with a current page: ${route.path}`);
  if (redirects.some(item => item.path === route.path)) errors.push(`retired route conflicts with redirect: ${route.path}`);
}

const forbiddenRoutePatterns = [
  /\bscenario=/u,
  /(?:intent|service)=(?:bazi|mingshu|bond)(?:&|'|"|$)/u,
  /chatUrl\([^)]*,\s*['"](?:bazi|mingshu|bond)['"]/u,
];
const sourceFiles = [];
function walk(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      if (['node_modules', 'dist', 'generated', 'coverage'].includes(entry.name)) continue;
      walk(full);
    } else if (/\.(?:ts|vue|astro|go|caddy)$/.test(entry.name)) sourceFiles.push(full);
  }
}
walk(path.join(root, 'app/src'));
walk(path.join(root, 'src'));
walk(path.join(root, '../internal/http'));
const deployCaddy = process.env.LIKI_DEPLOY_CADDYFILE ?? '/home/sq/wsp/liki/liki-deploy/caddy/Caddyfile';
if (fs.existsSync(deployCaddy)) sourceFiles.push(deployCaddy);

for (const file of sourceFiles) {
  const source = fs.readFileSync(file, 'utf8');
  for (const pattern of forbiddenRoutePatterns) {
    if (pattern.test(source)) errors.push(`retired route found in ${path.relative(root, file)}`);
  }
}

if (errors.length) {
  console.error(`✗ route contract has ${errors.length} issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}
console.log(`✓ route contracts are consistent (${intents.intents.length} intents, ${appRoutes.length} app routes)`);
