import { apiFetch } from './client'

export interface PriceSnapshot {
  id: string
  currency: string
  price_fiat: number
  source: string
  timestamp: string
}

export const listPrices = (currency: string, from?: string, to?: string) => {
  const params = new URLSearchParams({ currency })
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  return apiFetch<PriceSnapshot[]>(`/prices?${params}`)
}

export const createPrice = (data: {
  currency: string
  price_fiat: number
  source?: string
  timestamp: number
}) => apiFetch<PriceSnapshot>('/prices', { method: 'POST', body: JSON.stringify(data) })
