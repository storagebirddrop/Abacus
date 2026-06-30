import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { PriceSnapshot } from '../api/prices'

vi.mock('../api/prices', () => ({
  listPrices: vi.fn(),
  createPrice: vi.fn(),
}))

import { listPrices, createPrice } from '../api/prices'
import PricesPage from './PricesPage'

const listMock = listPrices as unknown as Mock
const createMock = createPrice as unknown as Mock

function price(over: Partial<PriceSnapshot> = {}): PriceSnapshot {
  return {
    id: 'p1', currency: 'EUR', price_fiat: 3_000_000, source: 'manual',
    timestamp: '2024-01-02T00:00:00Z', ...over,
  }
}

beforeEach(() => {
  listMock.mockReset()
  createMock.mockReset()
})
afterEach(() => vi.restoreAllMocks())

describe('PricesPage', () => {
  it('lists price snapshots with formatted amounts', async () => {
    listMock.mockResolvedValue([price()])
    render(<PricesPage />)
    // 3_000_000 cents → 30,000.00
    expect(await screen.findByText('30,000.00')).toBeInTheDocument()
    expect(screen.getByText('manual')).toBeInTheDocument()
    expect(listMock).toHaveBeenCalledWith('EUR')
  })

  it('shows an empty state when there are no prices', async () => {
    listMock.mockResolvedValue([])
    render(<PricesPage />)
    expect(await screen.findByText('No price snapshots')).toBeInTheDocument()
  })

  it('sorts by price when the Price header is toggled', async () => {
    listMock.mockResolvedValue([
      price({ id: 'p1', price_fiat: 3_000_000, timestamp: '2024-01-02T00:00:00Z' }),
      price({ id: 'p2', price_fiat: 1_000_000, timestamp: '2024-02-02T00:00:00Z' }),
    ])
    render(<PricesPage />)
    await screen.findByText('30,000.00')

    // Toggle to sort by price ascending → 10,000.00 row comes first.
    await userEvent.click(screen.getByRole('button', { name: /^Price/ }))
    const prices = screen.getAllByText(/,\d{3}\.00$/).map((el) => el.textContent)
    expect(prices[0]).toBe('10,000.00')
  })

  it('adds a price via the dialog (cents conversion + currency)', async () => {
    listMock.mockResolvedValue([])
    createMock.mockResolvedValue(price())
    render(<PricesPage />)
    await screen.findByText('No price snapshots')

    await userEvent.click(screen.getByRole('button', { name: 'Add Price' }))

    // Labels aren't associated (no htmlFor), and type=date/number play poorly
    // with userEvent.type; select by attribute/placeholder and set directly.
    const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement
    fireEvent.change(dateInput, { target: { value: '2024-01-02' } })
    fireEvent.change(screen.getByPlaceholderText('30000.00'), { target: { value: '30000.00' } })
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(createMock).toHaveBeenCalledTimes(1))
    expect(createMock).toHaveBeenCalledWith(
      expect.objectContaining({ currency: 'EUR', price_fiat: 3_000_000, source: 'manual' }),
    )
    expect(typeof createMock.mock.calls[0][0].timestamp).toBe('number')
  })
})
