import { apiFetch } from './client'

export interface Wallet {
  id: string
  name: string
  descriptor: string
  fingerprint: string
  type: string
  network: string
  created_at: string
}

export interface Transaction {
  id: string
  wallet_id: string
  txid: string
  block_height: number
  block_time: string
  fee_sats: number
  confirmed: boolean
}

export interface ImportJob {
  id: string
  wallet_id: string
  source: string
  filename: string
  status: 'pending' | 'running' | 'done' | 'failed'
  records_imported: number
  error_message: string
  started_at: string
  finished_at: string
}

export const listWallets = () => apiFetch<Wallet[]>('/wallets')

export const getWallet = (id: string) => apiFetch<Wallet>(`/wallets/${id}`)

export const createWallet = (data: { name: string; descriptor: string }) =>
  apiFetch<Wallet>('/wallets', {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const deleteWallet = (id: string) =>
  apiFetch<void>(`/wallets/${id}`, { method: 'DELETE' })

export interface TxQuery {
  page?: number
  limit?: number
  search?: string
  status?: '' | 'confirmed' | 'pending'
  sort?: 'date' | 'fee'
  dir?: 'asc' | 'desc'
}

export interface TxPage {
  data: Transaction[]
  total: number
  page: number
  limit: number
}

export const listTransactions = (walletID: string, q: TxQuery = {}) => {
  const params = new URLSearchParams()
  params.set('page', String(q.page ?? 1))
  params.set('limit', String(q.limit ?? 50))
  if (q.search) params.set('search', q.search)
  if (q.status) params.set('status', q.status)
  if (q.sort) params.set('sort', q.sort)
  if (q.dir) params.set('dir', q.dir)
  return apiFetch<TxPage>(`/wallets/${walletID}/transactions?${params.toString()}`)
}

export const importWallet = (walletID: string, file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return apiFetch<ImportJob>(`/wallets/${walletID}/import`, {
    method: 'POST',
    headers: {},
    body: fd,
  })
}

export const getImportJob = (jobID: string) =>
  apiFetch<ImportJob>(`/import-jobs/${jobID}`)

export const listImportJobs = (walletID: string) =>
  apiFetch<ImportJob[]>(`/wallets/${walletID}/import-jobs`)
