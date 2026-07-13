import { apiFetch } from './client'

export interface WalletSummary {
  wallet_id: string
  wallet_name: string
  method?: string
  fiat_currency?: string
  total_sats: number
  total_cost_fiat: number
  unrealised_gain_fiat: number
  realised_gain_fiat: number
}

export interface PortfolioSummary {
  wallet_count: number
  total_sats: number
  total_cost_fiat: number
  unrealised_gain_fiat: number
  realised_gain_fiat: number
  wallets: WalletSummary[]
  computed_at: string
}

export const getPortfolioSummary = () =>
  apiFetch<PortfolioSummary>('/portfolio/summary')

export interface PortfolioHistoryPoint {
  date: string // YYYY-MM-DD
  total_sats: number
  fiat_value: number // cents; 0 if no price known for that date
}

export const getPortfolioHistory = (currency: string, days = 180) =>
  apiFetch<PortfolioHistoryPoint[]>(`/portfolio/history?currency=${currency}&days=${days}`)
