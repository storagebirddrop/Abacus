import { afterEach, describe, expect, it, vi, type Mock } from 'vitest'

vi.mock('./client', () => ({ apiFetch: vi.fn().mockResolvedValue({}) }))

import { apiFetch } from './client'
import {
  listWallets,
  getWallet,
  createWallet,
  deleteWallet,
  listTransactions,
  importWallet,
  getImportJob,
  listImportJobs,
} from './wallets'

const mock = apiFetch as unknown as Mock

afterEach(() => mock.mockClear())

describe('wallets API contract', () => {
  it('listWallets → GET /wallets', () => {
    listWallets()
    expect(mock).toHaveBeenCalledWith('/wallets')
  })

  it('getWallet → GET /wallets/{id}', () => {
    getWallet('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1')
  })

  it('createWallet → POST /wallets with JSON body', () => {
    createWallet({ name: 'Cold', descriptor: 'wpkh(xpub...)' })
    expect(mock).toHaveBeenCalledWith('/wallets', {
      method: 'POST',
      body: JSON.stringify({ name: 'Cold', descriptor: 'wpkh(xpub...)' }),
    })
  })

  it('deleteWallet → DELETE /wallets/{id}', () => {
    deleteWallet('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1', { method: 'DELETE' })
  })

  it('listTransactions encodes pagination params', () => {
    listTransactions('w1', 50, 100)
    expect(mock).toHaveBeenCalledWith('/wallets/w1/transactions?limit=50&offset=100')
  })

  it('importWallet posts FormData and does NOT force a JSON content-type', () => {
    const file = new File(['{}'], 'wallet.json', { type: 'application/json' })
    importWallet('w1', file)
    const [path, init] = mock.mock.calls[0]
    expect(path).toBe('/wallets/w1/import')
    expect(init.method).toBe('POST')
    expect(init.body).toBeInstanceOf(FormData)
    expect((init.body as FormData).get('file')).toBe(file)
    // Must clear the default JSON header so fetch can set the multipart boundary.
    expect(init.headers).toEqual({})
  })

  it('getImportJob / listImportJobs hit the right paths', () => {
    getImportJob('job1')
    expect(mock).toHaveBeenCalledWith('/import-jobs/job1')
    listImportJobs('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/import-jobs')
  })
})
