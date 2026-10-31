#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..');
const readJson = value => JSON.parse(fs.readFileSync(value, 'utf8'));
const intents = readJson(path.join(root, 'shared/config/intents.json'));
const marketing = readJson(path.join(root, 'routes/marketing.contract.json'));
const app = readJson(path.join(root, 'routes/app.contract.json'));
const expectedLocales = ['zh-Hans', 'zh-Hant', 'en'];
const locales = Object.fromEntries(expectedLocales?.map(locale => [
  locale,
  readJson(path.join(root, 'app/src/locales', `${locale}.json`)),
]) ?? []);
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

for (const route of app.routes ?? []) {
  if (typeof route !== 'string' || !route.startsWith('/app')) errors.push(`invalid app route: ${route}`);
}
const appRouteCounts = new Map();
for (const route of app.routes ?? []) appRouteCounts.set(route, (appRouteCounts.get(route) ?? 0) + 1);
for (const [route, count] of appRouteCounts) {
  if (count > 1) errors.push(`duplicate app route: ${route}`);
}

const consultationRoute = '/app/consultation';
const consultationDefaultRoute = '/app';
const classicDefaultRoute = '/app/classic';
const compatibilityFlowRoute = '/app/flow';
const meRoute = '/app/me';
function localeValue(locale, key) {
  return key.split('.').reduce((value, part) => (value == null ? value : value[part]), locales[locale]);
}
if (appRouteCounts.get(consultationRoute) !== 1) {
  errors.push(`app contract must contain ${consultationRoute} exactly once`);
}
if (appRouteCounts.get(meRoute) !== 1) {
  errors.push(`app contract must contain ${meRoute} exactly once`);
}
if (appRouteCounts.has(compatibilityFlowRoute)) {
  errors.push(`implementation route must not be public: ${compatibilityFlowRoute}`);
}
if (app.featureFlags?.appFlowRedesign !== 'VITE_FEATURE_APP_FLOW_REDESIGN') {
  errors.push(`app feature flag must be VITE_FEATURE_APP_FLOW_REDESIGN`);
}

const expectedExperiences = {
  classic: {
    defaultRoute: classicDefaultRoute,
    rollbackRoute: '/app',
    status: 'maintenance',
  },
  consultation: {
    defaultRoute: consultationDefaultRoute,
    compatibilityAliasRoute: consultationRoute,
    status: 'preview',
    featureFlag: 'VITE_FEATURE_APP_FLOW_REDESIGN',
  },
};
if (JSON.stringify(app.experiences ?? {}) !== JSON.stringify(expectedExperiences)) {
  errors.push('app experiences must declare the consultation-default and classic rollback routes');
}

const expectedClassicSuffixes = [
  '',
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
];
const expectedClassicRoutes = expectedClassicSuffixes.map(suffix => `/app/classic${suffix}`);
const consultationAliasRedirect = {
  from: consultationRoute,
  to: consultationDefaultRoute,
  preserve: ['query', 'hash'],
};
const consultationRedirectTargets = {
  '/app/hepan': '/app?group=natal&intent=hepan',
  '/app/divination': '/app?group=divination&intent=divination',
  '/app/fengshui': '/app?group=fengshui&intent=fengshui',
};
const expectedLegacyRedirects = [
  consultationAliasRedirect,
  ...expectedClassicSuffixes.slice(1).map((suffix) => {
    const from = `/app${suffix}`;
    const consultationTarget = consultationRedirectTargets[from];
    return {
      from,
      to: consultationTarget ?? `/app/classic${suffix}`,
      preserve: ['query', 'hash'],
    };
  }),
]
expectedLegacyRedirects.push(
  {
    from: '/app/liunian',
    to: '/app?group=natal&intent=liunian',
    preserve: ['query', 'hash'],
  },
  {
    from: '/app/naming',
    to: '/app?group=naming&intent=naming',
    preserve: ['query', 'hash'],
  },
);
expectedLegacyRedirects.sort((left, right) => left.from.localeCompare(right.from));

const expectedRollout = {
  previewDefault: true,
  rollback: 'Set VITE_FEATURE_APP_FLOW_REDESIGN=false; /app returns to Classic.',
  cleanupStatus: 'single product; four intent groups remain consultation UI state, not product boundaries',
};
if (JSON.stringify(app.rollout ?? {}) !== JSON.stringify(expectedRollout)) {
  errors.push('app rollout must keep an explicit rollback contract and compatibility-owned cleanup status');
}
for (const route of expectedClassicRoutes) {
  if (appRouteCounts.get(route) !== 1) errors.push(`classic experience route missing or duplicated: ${route}`);
}
if (JSON.stringify(app.experienceRedirects ?? []) !== JSON.stringify(expectedLegacyRedirects)) {
  errors.push('legacy app redirects must preserve the complete Classic IA, query, and hash');
}

const queryContracts = app.queryContracts ?? {};
if (JSON.stringify(Object.keys(queryContracts)) !== JSON.stringify([consultationDefaultRoute])) {
  errors.push(`queryContracts must contain ${consultationDefaultRoute} only`);
}
const flowQuery = queryContracts[consultationDefaultRoute] ?? {};
const expectedFlowQueryKeys = ['group', 'intent', 'service', 'seed'];
if (JSON.stringify(Object.keys(flowQuery)) !== JSON.stringify(expectedFlowQueryKeys)) {
  errors.push(`consultation query contract must contain ${expectedFlowQueryKeys.join(', ')}`);
}
for (const key of ['intent', 'service', 'seed']) {
  if (flowQuery[key]?.type !== 'intentId') errors.push(`flow ${key} query must be an intentId`);
}
if (flowQuery.service?.policy !== 'legacy marketing alias for intent') {
  errors.push('flow service query must remain a legacy alias for intent');
}
if (flowQuery.seed?.policy !== 'legacy app alias for intent') {
  errors.push('flow seed query must remain a legacy alias for intent');
}
if (flowQuery.group?.type !== 'enum') errors.push('flow group query must be an intent-group enum');
for (const key of expectedFlowQueryKeys) {
  if (flowQuery[key]?.required !== false) errors.push(`flow ${key} query must be optional`);
}
if (typeof localeValue('zh-Hans', 'router.consultation') !== 'string' || typeof localeValue('zh-Hant', 'router.consultation') !== 'string' || typeof localeValue('en', 'router.consultation') !== 'string') {
  errors.push('router.consultation label is missing from a locale catalogue');
}
if (typeof localeValue('zh-Hans', 'router.me') !== 'string' || typeof localeValue('zh-Hant', 'router.me') !== 'string' || typeof localeValue('en', 'router.me') !== 'string') {
  errors.push('router.me label is missing from a locale catalogue');
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
      if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === 'generated') continue;
      walk(full);
    } else if (/\.(?:ts|vue|astro|go|caddy)$/.test(entry.name)) sourceFiles.push(full);
  }
}
walk(path.join(root, 'app/src'));
walk(path.join(root, 'src'));
walk(path.join(root, '../internal/http'));
for (const file of ['/home/sq/wsp/liki/liki-deploy/caddy/Caddyfile']) {
  if (fs.existsSync(file)) sourceFiles.push(file);
}
for (const file of sourceFiles) {
  const source = fs.readFileSync(file, 'utf8');
  for (const pattern of forbiddenRoutePatterns) {
    if (pattern.test(source)) {
      errors.push(`retired route found in ${path.relative(root, file)}`);
    }
  }
}

if (errors.length) {
  console.error(`✗ route contract has ${errors.length} issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}
console.log(`✓ route contracts are consistent (${intents.intents.length} intents, ${app.routes.length} app routes)`);
