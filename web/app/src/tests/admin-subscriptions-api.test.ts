import { describe, expect, it, vi } from 'vitest'
import { fetchAdminSubscriptions } from '@/api/admin'

describe('admin subscriptions api', () => {
  it('sends status and search filters and unwraps the page envelope', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: {
        subscriptions: [{ id: 'sub_1', email: 'ops@example.com', status: 'on_hold' }],
        page: 2,
        page_size: 50,
        total: 51,
        has_more: true,
      },
    }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    try {
      const result = await fetchAdminSubscriptions({ status: 'on_hold', search: 'sub_1' }, 2)

      expect(fetchMock.mock.calls[0][0]).toBe(
        '/api/admin/sales/subscriptions?status=on_hold&search=sub_1&page=2&page_size=50',
      )
      expect(result.items).toHaveLength(1)
      expect(result.meta).toMatchObject({ page: 2, page_size: 50, total: 51, has_more: true })
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('omits empty filters so the first page stays a bare request', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { subscriptions: [] },
    }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    try {
      const result = await fetchAdminSubscriptions({})

      expect(fetchMock.mock.calls[0][0]).toBe('/api/admin/sales/subscriptions?page=1&page_size=50')
      expect(result.items).toEqual([])
      expect(result.meta).toMatchObject({ page: 1, page_size: 50, total: 0, has_more: false })
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
