import { apiFetch } from './client'

export interface AccountingSummary {
  wallet_id: string
  method: string
  fiat_currency: string
  total_cost_sats: number
  total_cost_fiat: number
  unrealised_gain_fiat: number
  realised_gain_fiat: number
  computed_at: string
}

export interface CostBasisRecord {
  id: string
  wallet_id: string
  txid: string
  vout: number
  acquired_at: string
  cost_sats: number
  cost_fiat: number
  fiat_currency: string
  method: string
  disposed_at: string | null
  proceeds_fiat: number | null
  gain_fiat: number | null
}

export const runAccounting = (
  walletID: string,
  method: 'fifo' | 'avgcost',
  currency: string
) =>
  apiFetch<AccountingSummary>(`/wallets/${walletID}/accounting/run`, {
    method: 'POST',
    body: JSON.stringify({ method, currency }),
  })

export const getAccountingSummary = (walletID: string) =>
  apiFetch<AccountingSummary>(`/wallets/${walletID}/accounting/summary`)

export const listCostBasis = (walletID: string) =>
  apiFetch<CostBasisRecord[]>(`/wallets/${walletID}/accounting/cost-basis`)
