import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { ToastProvider } from '../components/Toast'
import { ConfirmProvider } from '../components/ConfirmDialog'
import type { Wallet } from '../api/wallets'

vi.mock('../api/wallets', () => ({
  listWallets: vi.fn(),
  createWallet: vi.fn(),
  importWallet: vi.fn(),
}))

import { listWallets, createWallet } from '../api/wallets'
import WalletsPage from './WalletsPage'

const listMock = listWallets as unknown as Mock
const createMock = createWallet as unknown as Mock

function wallet(over: Partial<Wallet> = {}): Wallet {
  return {
    id: 'w1', name: 'Cold Storage', descriptor: 'wpkh(xpub)', fingerprint: 'abcd1234',
    type: 'singlesig', network: 'mainnet', created_at: '2024-01-01T00:00:00Z', ...over,
  }
}

function renderPage() {
  return render(
    <MemoryRouter>
      <ToastProvider>
        <ConfirmProvider>
          <WalletsPage />
        </ConfirmProvider>
      </ToastProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  listMock.mockReset()
  createMock.mockReset()
})
afterEach(() => vi.restoreAllMocks())

describe('WalletsPage', () => {
  it('renders the wallet list', async () => {
    listMock.mockResolvedValue([wallet(), wallet({ id: 'w2', name: 'Hot Wallet' })])
    renderPage()
    expect(await screen.findByText('Cold Storage')).toBeInTheDocument()
    expect(screen.getByText('Hot Wallet')).toBeInTheDocument()
  })

  it('shows an empty state when there are no wallets', async () => {
    listMock.mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('No wallets yet')).toBeInTheDocument()
  })

  it('creates a wallet via the dialog and reloads the list', async () => {
    listMock.mockResolvedValue([])
    createMock.mockResolvedValue(wallet())
    renderPage()
    await screen.findByText('No wallets yet')

    await userEvent.click(screen.getByRole('button', { name: 'Add Wallet' }))
    await userEvent.type(screen.getByPlaceholderText('My Bitcoin Wallet'), 'Cold Storage')
    await userEvent.click(screen.getByRole('button', { name: 'Add Wallet' }))

    expect(createMock).toHaveBeenCalledWith({ name: 'Cold Storage', descriptor: '' })
    // load() runs once on mount and again after creation.
    expect(listMock).toHaveBeenCalledTimes(2)
  })

  it('filters the list via the search box', async () => {
    listMock.mockResolvedValue([wallet(), wallet({ id: 'w2', name: 'Hot Wallet', fingerprint: 'ffff' })])
    renderPage()
    await screen.findByText('Cold Storage')

    await userEvent.type(screen.getByRole('searchbox', { name: /search wallets/i }), 'hot')
    expect(screen.getByText('Hot Wallet')).toBeInTheDocument()
    expect(screen.queryByText('Cold Storage')).not.toBeInTheDocument()
  })

  it('sorts by name and reverses on header click', async () => {
    listMock.mockResolvedValue([wallet({ id: 'w2', name: 'Zebra' }), wallet({ id: 'w1', name: 'Alpha' })])
    renderPage()
    await screen.findByText('Alpha')

    const names = () => screen.getAllByRole('link').map((a) => a.textContent)
    expect(names()).toEqual(['Alpha', 'Zebra']) // default asc

    await userEvent.click(screen.getByRole('button', { name: /^Name/ }))
    expect(names()).toEqual(['Zebra', 'Alpha']) // desc after toggle
  })
})
