import { apiFetch } from './client'

export interface SyncJob {
  id: string
  wallet_id: string
  backend: string
  status: 'pending' | 'running' | 'done' | 'failed'
  addresses_scanned: number
  tx_found: number
  error_message?: string
  started_at: string
  finished_at?: string
}

export function startSync(walletID: string): Promise<{ job_id: string }> {
  return apiFetch(`/wallets/${walletID}/sync`, { method: 'POST' })
}

export function getSyncJob(jobID: string): Promise<SyncJob> {
  return apiFetch(`/sync-jobs/${jobID}`)
}

export function listSyncJobs(walletID: string): Promise<SyncJob[]> {
  return apiFetch(`/wallets/${walletID}/sync-jobs`)
}
