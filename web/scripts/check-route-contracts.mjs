#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..');
const readJson = value => JSON.parse(fs.readFileSync(value, 'utf8'));
const scenarios = readJson(path.join(root, 'shared/config/scenarios.json'));
const marketing = readJson(path.join(root, 'routes/marketing.contract.json'));
const app = readJson(path.join(root, 'routes/app.contract.json'));
const errors = [];

if (scenarios.version !== 1) errors.push('scenarios.version must be 1');
if (marketing.version !== 1) errors.push('marketing.version must be 1');
if (app.version !== 1) errors.push('app.version must be 1');

const expectedLocales = ['zh-Hans', 'zh-Hant', 'en'];
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
for (const scenario of scenarios.scenarios ?? []) {
  if (scenario.appPath && !app.routes.includes(scenario.appPath)) {
    errors.push(`scenario appPath missing from app contract: ${scenario.appPath}`);
  }
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
