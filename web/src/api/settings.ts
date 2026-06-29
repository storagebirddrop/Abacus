import { apiFetch } from './client'

export interface AppSettings {
  sync_enabled: boolean
  blockchain_backend: 'esplora' | 'electrum'
  esplora_url: string
  esplora_rate_ms: number
  electrum_host: string
  electrum_port: number
  electrum_tls: boolean
}

export const getSettings = () =>
  apiFetch<AppSettings>('/settings')

export const updateSettings = (data: Partial<AppSettings>) =>
  apiFetch<AppSettings>('/settings', {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
