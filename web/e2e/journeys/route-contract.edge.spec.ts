// 这些用例必须跑在 Caddy edge 上：assembly、预发布 origin 或生产。
// 纯静态 dist server 不执行 generated Caddy redirects，因此不能作为本契约环境。
import { expect, test } from '../fixtures'
import marketingContract from '../../routes/marketing.contract.json' with { type: 'json' }

test.describe('intent route contract on edge', () => {
  for (const redirect of marketingContract.bareRedirects) {
    test(`${redirect.path} redirects to the canonical slashed page`, async ({ request }) => {
      const response = await request.get(redirect.path, {
        headers: { 'Accept-Language': 'zh-Hant' },
        maxRedirects: 0,
      })
      expect(response.status()).toBe(redirect.status)
      expect(response.headers().location).toBe(`/zh-Hant/${redirect.slug}/`)
    })
  }
})
