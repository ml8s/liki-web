// journey 共享的 fixture。
//
// 契约：journey 必须显式声明它依赖的接口（用例里的 page.route）。没声明的 /api 请求
// 会被拦下、判失败并列出 URL——否则它会落到 dev/preview 代理（本地没人监听）或真实
// 后端（docker）上，用例就可能靠环境变绿，而不是靠 mock。要测失败态（401/403/500）
// 就在用例里显式 mock 失败。
//
// 契约挂在 context 而不是 page 上：新开的标签页、弹窗同样受约束。
// Playwright 先匹配 page 路由，再回落到 context 路由，所以用例自己的 page.route
// 照常生效；live-system 打的是真实后端，不装这层契约。
import { test as base, expect } from '@playwright/test';

// 匹配用 URL 谓词而不是 `**/api/**` 这类 glob：只认 HTTP API 命名空间，
// 不会把 dev server 的 /src/api/*.ts 模块请求当成 API 调用。
const isApiRequest = (url: URL) => url.pathname.startsWith('/api/');

const test = base.extend({
  context: async ({ context }, use) => {
    const undeclared = new Set();
    if (!process.env.LIVE_SYSTEM_E2E) {
      await context.route(isApiRequest, route => {
        undeclared.add(`${route.request().method()} ${route.request().url()}`);
        return route.abort('failed');
      });
    }
    await use(context);
    if (undeclared.size > 0) {
      throw new Error(
        `未声明的 API 调用（请在用例里用 page.route 声明）：\n  ${[...undeclared].join('\n  ')}`,
      );
    }
  },
});

export { test, expect };
