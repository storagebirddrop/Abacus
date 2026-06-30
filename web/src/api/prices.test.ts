import { afterEach, describe, expect, it, vi, type Mock } from 'vitest'

vi.mock('./client', () => ({ apiFetch: vi.fn().mockResolvedValue({}) }))

import { apiFetch } from './client'
import { listPrices, createPrice } from './prices'

const mock = apiFetch as unknown as Mock

afterEach(() => mock.mockClear())

describe('prices API contract', () => {
  it('listPrices with only a currency', () => {
    listPrices('EUR')
    expect(mock).toHaveBeenCalledWith('/prices?currency=EUR')
  })

  it('listPrices includes from/to when provided', () => {
    listPrices('USD', '2024-01-01', '2024-12-31')
    const path = mock.mock.calls[0][0] as string
    expect(path).toContain('currency=USD')
    expect(path).toContain('from=2024-01-01')
    expect(path).toContain('to=2024-12-31')
  })

  it('createPrice → POST /prices with JSON body', () => {
    createPrice({ currency: 'EUR', price_fiat: 6_000_000, timestamp: 1700000000 })
    expect(mock).toHaveBeenCalledWith('/prices', {
      method: 'POST',
      body: JSON.stringify({ currency: 'EUR', price_fiat: 6_000_000, timestamp: 1700000000 }),
    })
  })
})
