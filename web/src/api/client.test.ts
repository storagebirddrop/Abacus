import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiFetch, ApiError } from './client'

function mockFetch(impl: typeof fetch) {
  vi.stubGlobal('fetch', vi.fn(impl))
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('apiFetch', () => {
  it('prefixes /api/v1 and returns parsed JSON on success', async () => {
    mockFetch(async () =>
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )
    const data = await apiFetch<{ ok: boolean }>('/wallets')
    expect(data).toEqual({ ok: true })
    expect(fetch).toHaveBeenCalledWith('/api/v1/wallets', expect.any(Object))
  })

  it('sends a JSON Content-Type header by default', async () => {
    mockFetch(async () => new Response('{}', { status: 200 }))
    await apiFetch('/wallets')
    const init = (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0][1]
    expect(init.headers['Content-Type']).toBe('application/json')
  })

  it('throws ApiError with the server error message on non-2xx', async () => {
    mockFetch(async () =>
      new Response(JSON.stringify({ error: 'wallet not found' }), { status: 404 }),
    )
    await expect(apiFetch('/wallets/nope')).rejects.toMatchObject({
      status: 404,
      message: 'wallet not found',
    })
    await expect(apiFetch('/wallets/nope')).rejects.toBeInstanceOf(ApiError)
  })

  it('falls back to an HTTP status message when the body has no error field', async () => {
    mockFetch(async () => new Response('not json', { status: 500 }))
    await expect(apiFetch('/boom')).rejects.toMatchObject({
      status: 500,
      message: 'HTTP 500',
    })
  })
})
