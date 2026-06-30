import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import type { Wallet } from '../api/wallets'

vi.mock('../api/wallets', () => ({
  listWallets: vi.fn(),
  createWallet: vi.fn(),
  deleteWallet: vi.fn(),
}))

import { listWallets, createWallet, deleteWallet } from '../api/wallets'
import WalletsPage from './WalletsPage'

const listMock = listWallets as unknown as Mock
const createMock = createWallet as unknown as Mock
const deleteMock = deleteWallet as unknown as Mock

function wallet(over: Partial<Wallet> = {}): Wallet {
  return {
    id: 'w1', name: 'Cold Storage', descriptor: 'wpkh(xpub)', fingerprint: 'abcd1234',
    type: 'singlesig', network: 'mainnet', created_at: '2024-01-01T00:00:00Z', ...over,
  }
}

function renderPage() {
  return render(<MemoryRouter><WalletsPage /></MemoryRouter>)
}

beforeEach(() => {
  listMock.mockReset()
  createMock.mockReset()
  deleteMock.mockReset()
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

    await userEvent.click(screen.getByRole('button', { name: 'New Wallet' }))
    await userEvent.type(screen.getByPlaceholderText('My Bitcoin Wallet'), 'Cold Storage')
    await userEvent.type(
      screen.getByPlaceholderText('wpkh([fingerprint/path]xpub...)'),
      'wpkh(xpub123)',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() =>
      expect(createMock).toHaveBeenCalledWith({ name: 'Cold Storage', descriptor: 'wpkh(xpub123)' }),
    )
    // load() runs once on mount and again after creation.
    expect(listMock).toHaveBeenCalledTimes(2)
  })

  it('deletes a wallet after confirmation', async () => {
    listMock.mockResolvedValue([wallet()])
    deleteMock.mockResolvedValue(undefined)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderPage()

    const row = (await screen.findByText('Cold Storage')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(deleteMock).toHaveBeenCalledWith('w1'))
    await waitFor(() => expect(screen.queryByText('Cold Storage')).not.toBeInTheDocument())
  })

  it('does not delete when confirmation is dismissed', async () => {
    listMock.mockResolvedValue([wallet()])
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    renderPage()

    const row = (await screen.findByText('Cold Storage')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Delete' }))

    expect(deleteMock).not.toHaveBeenCalled()
    expect(screen.getByText('Cold Storage')).toBeInTheDocument()
  })
})
