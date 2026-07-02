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
