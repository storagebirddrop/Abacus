import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

vi.mock('../../api/wallets', () => ({
  importWallet: vi.fn(),
  getImportJob: vi.fn(),
  listImportJobs: vi.fn(),
}))
vi.mock('../../api/sync', () => ({
  startSync: vi.fn(),
  getSyncJob: vi.fn(),
  listSyncJobs: vi.fn(),
}))

import { importWallet, getImportJob, listImportJobs } from '../../api/wallets'
import { startSync, getSyncJob, listSyncJobs } from '../../api/sync'
import { ImportTab } from './ImportTab'
import { SyncPanel } from './SyncPanel'

const importMock = importWallet as unknown as Mock
const getImportMock = getImportJob as unknown as Mock
const listImportMock = listImportJobs as unknown as Mock
const startSyncMock = startSync as unknown as Mock
const getSyncMock = getSyncJob as unknown as Mock
const listSyncMock = listSyncJobs as unknown as Mock

beforeEach(() => {
  vi.useFakeTimers()
  ;[importMock, getImportMock, listImportMock, startSyncMock, getSyncMock, listSyncMock].forEach((m) => m.mockReset())
})
afterEach(() => {
  vi.useRealTimers()
})

describe('ImportTab polling', () => {
  it('polls the import job to completion and reports records imported', async () => {
    listImportMock.mockResolvedValue([])
    importMock.mockResolvedValue({ id: 'job12345', status: 'running', filename: 'w.json' })
    getImportMock.mockResolvedValue({ id: 'job12345', status: 'done', records_imported: 5 })

    render(<ImportTab walletID="w1" />)
    await advance(0) // flush initial listImportJobs

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    fireEvent.change(fileInput, { target: { files: [new File(['{}'], 'w.json')] } })
    fireEvent.submit(fileInput.closest('form')!)

    await advance(0) // importWallet resolves → "Job started", poll armed
    expect(screen.getByText(/Job started/i)).toBeInTheDocument()

    await advance(2000) // poll fires → getImportJob → done
    expect(getImportMock).toHaveBeenCalledWith('job12345')
    expect(screen.getByText(/Done — 5 records imported/i)).toBeInTheDocument()
  })
})

describe('SyncPanel polling', () => {
  it('polls the sync job to completion and reports the summary', async () => {
    listSyncMock.mockResolvedValue([])
    startSyncMock.mockResolvedValue({ job_id: 'syncjob1' })
    getSyncMock.mockResolvedValue({
      id: 'syncjob1', status: 'done', tx_found: 7, addresses_scanned: 20, backend: 'esplora',
    })

    render(<SyncPanel walletID="w1" />)
    await advance(0) // flush initial listSyncJobs

    fireEvent.click(screen.getByRole('button', { name: 'Sync from Blockchain' }))

    await advance(0) // startSync resolves → "Sync job started", poll armed
    expect(screen.getByText(/Sync job started/i)).toBeInTheDocument()

    await advance(2000) // poll fires → getSyncJob → done
    expect(getSyncMock).toHaveBeenCalledWith('syncjob1')
    expect(screen.getByText(/Done — 7 transactions, 20 addresses scanned/i)).toBeInTheDocument()
  })
})
