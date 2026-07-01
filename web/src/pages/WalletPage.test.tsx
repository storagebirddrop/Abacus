import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import type { Wallet, Transaction, ImportJob } from '../api/wallets'
import type { AccountingSummary } from '../api/accounting'

vi.mock('../api/wallets', () => ({
  getWallet: vi.fn(),
  listTransactions: vi.fn(),
  importWallet: vi.fn(),
  getImportJob: vi.fn(),
  listImportJobs: vi.fn(),
}))
vi.mock('../api/accounting', () => ({
  getAccountingSummary: vi.fn(),
  listCostBasis: vi.fn(),
  runAccounting: vi.fn(),
}))
vi.mock('../api/sync', () => ({
  startSync: vi.fn(),
  getSyncJob: vi.fn(),
  listSyncJobs: vi.fn(),
}))

import { getWallet, listTransactions, importWallet, listImportJobs } from '../api/wallets'
import { getAccountingSummary, listCostBasis, runAccounting } from '../api/accounting'
import { startSync, listSyncJobs } from '../api/sync'
import WalletPage from './WalletPage'

const m = {
  getWallet: getWallet as unknown as Mock,
  listTransactions: listTransactions as unknown as Mock,
  importWallet: importWallet as unknown as Mock,
  listImportJobs: listImportJobs as unknown as Mock,
  getAccountingSummary: getAccountingSummary as unknown as Mock,
  listCostBasis: listCostBasis as unknown as Mock,
  runAccounting: runAccounting as unknown as Mock,
  startSync: startSync as unknown as Mock,
  listSyncJobs: listSyncJobs as unknown as Mock,
}

const wallet: Wallet = {
  id: 'w1', name: 'Cold Storage', descriptor: 'wpkh(xpub)', fingerprint: 'abcd1234',
  type: 'singlesig', network: 'mainnet', created_at: '2024-01-01T00:00:00Z',
}

function tx(over: Partial<Transaction> = {}): Transaction {
  return {
    id: 't1', wallet_id: 'w1', txid: 'a'.repeat(64), block_height: 800000,
    block_time: '2024-03-01T00:00:00Z', fee_sats: 250, confirmed: true, ...over,
  }
}

function renderAt() {
  return render(
    <MemoryRouter initialEntries={['/wallets/w1']}>
      <Routes>
        <Route path="/wallets/:id" element={<WalletPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  Object.values(m).forEach((fn) => fn.mockReset())
  // Sensible defaults so any tab can mount without unhandled rejections.
  m.getWallet.mockResolvedValue(wallet)
  m.listTransactions.mockResolvedValue({ data: [], total: 0, page: 1, limit: 50 })
  m.getAccountingSummary.mockResolvedValue(null)
  m.listCostBasis.mockResolvedValue([])
  m.runAccounting.mockResolvedValue({} as AccountingSummary)
  m.listImportJobs.mockResolvedValue([])
  m.importWallet.mockResolvedValue({ id: 'job12345', status: 'running' } as ImportJob)
  m.listSyncJobs.mockResolvedValue([])
  m.startSync.mockResolvedValue({ job_id: 'syncjob1' })
})
afterEach(() => vi.restoreAllMocks())

describe('WalletPage', () => {
  it('renders the wallet header and defaults to the Transactions tab', async () => {
    renderAt()
    expect(await screen.findByRole('heading', { name: 'Cold Storage' })).toBeInTheDocument()
    await waitFor(() =>
      expect(m.listTransactions).toHaveBeenCalledWith('w1', {
        page: 1, limit: 50, search: '', status: '', sort: 'date', dir: 'desc',
      }),
    )
  })

  it('shows the empty-transactions message', async () => {
    renderAt()
    expect(await screen.findByText(/No transactions/i)).toBeInTheDocument()
  })

  it('paginates transactions (Next advances the page)', async () => {
    m.listTransactions.mockResolvedValue({ data: [tx()], total: 120, page: 1, limit: 50 })
    renderAt()
    expect(await screen.findByText('Showing 1–50 of 120')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() =>
      expect(m.listTransactions).toHaveBeenCalledWith('w1', {
        page: 2, limit: 50, search: '', status: '', sort: 'date', dir: 'desc',
      }),
    )
  })

  it('runs accounting and renders the summary cards', async () => {
    m.runAccounting.mockResolvedValue({
      wallet_id: 'w1', method: 'fifo', fiat_currency: 'EUR',
      total_cost_sats: 0, total_cost_fiat: 30_000,
      unrealised_gain_fiat: 0, realised_gain_fiat: 0, computed_at: '',
    })
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Accounting' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Run Accounting' }))

    await waitFor(() => expect(m.runAccounting).toHaveBeenCalledWith('w1', 'fifo', 'EUR'))
    expect(await screen.findByText('Total Cost')).toBeInTheDocument()
    expect(screen.getByText('€300.00')).toBeInTheDocument()
  })

  it('lists existing import jobs on the Import tab', async () => {
    m.listImportJobs.mockResolvedValue([
      { id: 'j1', wallet_id: 'w1', source: 'sparrow', filename: 'export.json',
        status: 'done', records_imported: 12, error_message: '',
        started_at: '2024-01-01T00:00:00Z', finished_at: '' } as ImportJob,
    ])
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    expect(await screen.findByText('export.json')).toBeInTheDocument()
    expect(screen.getByText('done')).toBeInTheDocument()
  })

  it('uploads a file and reports the started job', async () => {
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['{}'], 'wallet.json', { type: 'application/json' })
    await userEvent.upload(fileInput, file)
    expect(fileInput.files?.[0]).toBe(file)

    await waitFor(() => expect(m.importWallet).toHaveBeenCalledWith('w1', file))
    expect(await screen.findByText(/Job started/i)).toBeInTheDocument()
  })

  it('starts a blockchain sync', async () => {
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Sync' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Sync from Blockchain' }))

    await waitFor(() => expect(m.startSync).toHaveBeenCalledWith('w1'))
    expect(await screen.findByText(/Sync job started/i)).toBeInTheDocument()
  })

  it('surfaces a wallet-load failure instead of failing silently', async () => {
    m.getWallet.mockRejectedValue(new Error('wallet not found'))
    renderAt()
    expect(await screen.findByRole('alert')).toHaveTextContent('wallet not found')
  })

  it('surfaces an accounting-load failure on the Accounting tab', async () => {
    m.getAccountingSummary.mockRejectedValue(new Error('boom'))
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Accounting' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('surfaces an import-history load failure on the Import tab', async () => {
    m.listImportJobs.mockRejectedValue(new Error('history down'))
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('history down')
  })

  it('surfaces a sync-history load failure on the Sync tab', async () => {
    m.listSyncJobs.mockRejectedValue(new Error('sync down'))
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Sync' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('sync down')
  })
})
