import { apiFetch } from './client'

export interface AppSettings {
  sync_enabled: boolean
  blockchain_backend: 'esplora' | 'electrum' | 'bitcoincore'
  esplora_url: string
  esplora_rate_ms: number
  electrum_host: string
  electrum_port: number
  electrum_tls: boolean
  bitcoincore_rpc_url: string
  bitcoincore_rpc_user: string
  /** Write-only: never returned by GET /settings, only accepted by PATCH. */
  bitcoincore_rpc_pass?: string
}

export const getSettings = () =>
  apiFetch<AppSettings>('/settings')

export const updateSettings = (data: Partial<AppSettings>) =>
  apiFetch<AppSettings>('/settings', {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
