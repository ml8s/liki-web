#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..');
const readJson = value => JSON.parse(fs.readFileSync(value, 'utf8'));
const scenarios = readJson(path.join(root, 'shared/config/scenarios.json'));
const domains = readJson(path.join(root, 'shared/config/domains.json'));
const marketing = readJson(path.join(root, 'routes/marketing.contract.json'));
const app = readJson(path.join(root, 'routes/app.contract.json'));
const expectedLocales = ['zh-Hans', 'zh-Hant', 'en'];
const locales = Object.fromEntries(expectedLocales?.map(locale => [
  locale,
  readJson(path.join(root, 'app/src/locales', `${locale}.json`)),
]) ?? []);
const errors = [];

if (scenarios.version !== 1) errors.push('scenarios.version must be 1');
if (domains.version !== 1) errors.push('domains.version must be 1');
if (marketing.version !== 1) errors.push('marketing.version must be 1');
if (app.version !== 1) errors.push('app.version must be 1');

if (JSON.stringify(marketing.locales) !== JSON.stringify(expectedLocales)) {
  errors.push('marketing.locales must equal zh-Hans, zh-Hant, en');
}

const scenarioIds = new Set();
const slugs = new Set();
const serviceParams = new Set();
const appPaths = new Set();
for (const scenario of scenarios.scenarios ?? []) {
  for (const key of ['id', 'expert', 'marketingSlug', 'serviceParam']) {
    if (typeof scenario[key] !== 'string' || scenario[key] === '') errors.push(`scenario.${key} is required`);
  }
  if (scenarioIds.has(scenario.id)) errors.push(`duplicate scenario id: ${scenario.id}`);
  if (slugs.has(scenario.marketingSlug)) errors.push(`duplicate marketing slug: ${scenario.marketingSlug}`);
  if (serviceParams.has(scenario.serviceParam)) errors.push(`duplicate service param: ${scenario.serviceParam}`);
  if (!scenarios.experts.includes(scenario.expert)) errors.push(`invalid expert: ${scenario.expert}`);
  if (scenario.appPath != null) {
    if (!scenario.appPath.startsWith('/app/')) errors.push(`appPath must start with /app/: ${scenario.appPath}`);
    if (appPaths.has(scenario.appPath)) errors.push(`duplicate appPath: ${scenario.appPath}`);
    appPaths.add(scenario.appPath);
  }
  scenarioIds.add(scenario.id);
  slugs.add(scenario.marketingSlug);
  serviceParams.add(scenario.serviceParam);
}

for (const route of app.routes ?? []) {
  if (typeof route !== 'string' || !route.startsWith('/app')) errors.push(`invalid app route: ${route}`);
}
const appRouteCounts = new Map();
for (const route of app.routes ?? []) appRouteCounts.set(route, (appRouteCounts.get(route) ?? 0) + 1);
for (const [route, count] of appRouteCounts) {
  if (count > 1) errors.push(`duplicate app route: ${route}`);
}

const scenarioIdSet = new Set(scenarios.scenarios.map(item => item.id));
const domainIds = new Set();
const domainSeeds = new Set();
const expectedDomains = ['natal', 'divination', 'fengshui', 'naming'];
const expectedZhHansDomainLabels = {
  natal: '命理',
  divination: '问卦',
  fengshui: '风水',
  naming: '起名',
};
function localeValue(locale, key) {
  return key.split('.').reduce((value, part) => (value == null ? value : value[part]), locales[locale]);
}
for (const domain of domains.domains ?? []) {
  for (const key of ['id', 'labelKey', 'baseSkill', 'hybridDefaultPath', 'profilePolicy']) {
    if (typeof domain[key] !== 'string' || domain[key] === '') {
      errors.push(`domain.${key} is required`);
    }
  }
  if (domainIds.has(domain.id)) errors.push(`duplicate domain id: ${domain.id}`);
  domainIds.add(domain.id);
  if (!scenarioIdSet.has(domain.baseSkill)) errors.push(`unknown domain baseSkill: ${domain.baseSkill}`);
  if (!['required', 'optional', 'partial'].includes(domain.profilePolicy)) {
    errors.push(`invalid domain profilePolicy: ${domain.profilePolicy}`);
  }
  if (!app.routes.includes(domain.hybridDefaultPath)) {
    errors.push(`domain hybrid path missing from app contract: ${domain.hybridDefaultPath}`);
  }
  if (!Array.isArray(domain.acceptedSeeds) || domain.acceptedSeeds.length === 0) {
    errors.push(`domain acceptedSeeds must be non-empty: ${domain.id}`);
  }
  if (!Array.isArray(domain.bridgeSkills) || domain.bridgeSkills.length === 0) {
    errors.push(`domain bridgeSkills must be non-empty: ${domain.id}`);
  }
  for (const seed of domain.acceptedSeeds ?? []) {
    if (!scenarioIdSet.has(seed)) errors.push(`unknown domain seed: ${seed}`);
    if (domainSeeds.has(seed)) errors.push(`seed belongs to multiple domains: ${seed}`);
    domainSeeds.add(seed);
  }
  const bridgeSkills = new Set(domain.bridgeSkills ?? []);
  for (const skill of bridgeSkills) {
    if (!domain.acceptedSeeds?.includes(skill)) {
      errors.push(`domain bridgeSkill is not an accepted seed: ${domain.id}/${skill}`);
    }
  }
  if (domain.baseSkill && !bridgeSkills.has(domain.baseSkill)) {
    errors.push(`domain baseSkill is not bridgeable: ${domain.id}`);
  }
  for (const locale of expectedLocales) {
    if (typeof localeValue(locale, domain.labelKey) !== 'string') {
      errors.push(`domain labelKey is missing from ${locale}: ${domain.id}/${domain.labelKey}`);
    }
  }
  if (localeValue('zh-Hans', domain.labelKey) !== expectedZhHansDomainLabels[domain.id]) {
    errors.push(`canonical zh-Hans domain label mismatch: ${domain.id}`);
  }
}
if (JSON.stringify([...domainIds]) !== JSON.stringify(expectedDomains)) {
  errors.push(`domains must equal ${expectedDomains.join(', ')}`);
}
for (const scenario of scenarios.scenarios) {
  if (!domainSeeds.has(scenario.id)) errors.push(`scenario is not mapped to a domain: ${scenario.id}`);
}
for (const scenario of scenarios.scenarios ?? []) {
  if (scenario.appPath && !app.routes.includes(scenario.appPath)) {
    errors.push(`scenario appPath missing from app contract: ${scenario.appPath}`);
  }
}

const flowRoute = '/app/flow';
if (appRouteCounts.get(flowRoute) !== 1) {
  errors.push(`app contract must contain ${flowRoute} exactly once`);
}
if (app.featureFlags?.appFlowRedesign !== 'VITE_FEATURE_APP_FLOW_REDESIGN') {
  errors.push(`app feature flag must be VITE_FEATURE_APP_FLOW_REDESIGN`);
}

const queryContracts = app.queryContracts ?? {};
if (JSON.stringify(Object.keys(queryContracts)) !== JSON.stringify([flowRoute])) {
  errors.push(`queryContracts must contain ${flowRoute} only`);
}
const flowQuery = queryContracts[flowRoute] ?? {};
const expectedFlowQueryKeys = ['domain', 'seed', 'service'];
if (JSON.stringify(Object.keys(flowQuery)) !== JSON.stringify(expectedFlowQueryKeys)) {
  errors.push(`flow query contract must contain ${expectedFlowQueryKeys.join(', ')}`);
}
if (JSON.stringify(flowQuery.domain?.values) !== JSON.stringify(expectedDomains) || flowQuery.domain?.default !== 'natal') {
  errors.push(`flow domain query must equal ${expectedDomains.join(', ')} with the natal default`);
}
for (const key of ['seed', 'service']) {
  if (flowQuery[key]?.type !== 'legacyScenarioId') errors.push(`flow ${key} query must be a legacyScenarioId`);
}
if (flowQuery.service?.policy !== 'legacy marketing alias for seed; public URLs remain unchanged') {
  errors.push('flow service query must remain a legacy alias for seed');
}
for (const key of expectedFlowQueryKeys) {
  if (flowQuery[key]?.required !== false) errors.push(`flow ${key} query must be optional`);
}
if (typeof localeValue('zh-Hans', 'router.flow') !== 'string' || typeof localeValue('zh-Hant', 'router.flow') !== 'string' || typeof localeValue('en', 'router.flow') !== 'string') {
  errors.push('router.flow label is missing from a locale catalogue');
}

const redirects = marketing.bareRedirects ?? [];
const retired = marketing.retired ?? [];
for (const route of redirects) {
  if (!scenarioIds.has(route.slug)) errors.push(`redirect references unknown scenario: ${route.slug}`);
  if (route.status !== 302) errors.push(`redirect must use 302: ${route.path}`);
}
for (const route of retired) {
  if (route.status !== 404) errors.push(`retired route must use 404: ${route.path}`);
  if (slugs.has(route.path.slice(1))) errors.push(`retired route conflicts with a current page: ${route.path}`);
  if (redirects.some(item => item.path === route.path)) errors.push(`retired route conflicts with redirect: ${route.path}`);
}

const forbiddenRoutePatterns = [
  /(?:scenario|service)=(?:bazi|mingshu|bond)(?:&|'|"|$)/u,
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
      errors.push(`retired scenario route found in ${path.relative(root, file)}`);
    }
  }
}

if (errors.length) {
  console.error(`✗ route contract has ${errors.length} issue(s):`);
  for (const error of errors) console.error(`  ✗ ${error}`);
  process.exit(1);
}
console.log(`✓ route contracts are consistent (${scenarios.scenarios.length} scenarios, ${app.routes.length} app routes)`);
