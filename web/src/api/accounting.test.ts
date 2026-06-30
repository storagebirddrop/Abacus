import { afterEach, describe, expect, it, vi, type Mock } from 'vitest'

vi.mock('./client', () => ({ apiFetch: vi.fn().mockResolvedValue({}) }))

import { apiFetch } from './client'
import { runAccounting, getAccountingSummary, listCostBasis } from './accounting'

const mock = apiFetch as unknown as Mock

afterEach(() => mock.mockClear())

describe('accounting API contract', () => {
  it('runAccounting → POST run with method + currency body', () => {
    runAccounting('w1', 'fifo', 'EUR')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/accounting/run', {
      method: 'POST',
      body: JSON.stringify({ method: 'fifo', currency: 'EUR' }),
    })
  })

  it('getAccountingSummary → GET summary', () => {
    getAccountingSummary('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/accounting/summary')
  })

  it('listCostBasis → GET cost-basis', () => {
    listCostBasis('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/accounting/cost-basis')
  })
})
