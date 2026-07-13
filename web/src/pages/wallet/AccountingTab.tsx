import { useEffect, useState } from 'react'
import {
  getAccountingSummary,
  listCostBasis,
  runAccounting,
  type AccountingMethod,
  type AccountingSummary,
  type CostBasisRecord,
} from '../../api/accounting'
import { Button } from '../../components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../components/ui/select'
import { cn } from '../../lib/utils'
import { ExportBar } from './ExportBar'

const CURRENCY_SYMBOLS: Record<string, string> = { EUR: '€', USD: '$', GBP: '£' }

function fmtCents(cents: number | null, currency: string) {
  if (cents === null || cents === 0) return '—'
  const sym = CURRENCY_SYMBOLS[currency] ?? currency + ' '
  return `${sym}${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}`
}

export function AccountingTab({ walletID }: { walletID: string }) {
  const [method, setMethod] = useState<AccountingMethod>('fifo')
  const [currency, setCurrency] = useState('EUR')
  const [summary, setSummary] = useState<AccountingSummary | null>(null)
  const [records, setRecords] = useState<CostBasisRecord[]>([])
  const [running, setRunning] = useState(false)
  const [error, setError] = useState('')
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    setLoadError('')
    Promise.all([getAccountingSummary(walletID), listCostBasis(walletID)])
      .then(([sum, recs]) => {
        setSummary(sum)
        setRecords(recs ?? [])
        if (sum?.fiat_currency) setCurrency(sum.fiat_currency)
      })
      .catch((err: unknown) =>
        setLoadError(err instanceof Error ? err.message : 'Failed to load accounting data'),
      )
  }, [walletID])

  async function handleRun(e: React.FormEvent) {
    e.preventDefault()
    setRunning(true)
    setError('')
    try {
      const sum = await runAccounting(walletID, method, currency)
      setSummary(sum)
      const recs = await listCostBasis(walletID)
      setRecords(recs ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="space-y-6">
      {loadError && (
        <p role="alert" className="text-sm text-destructive">{loadError}</p>
      )}

      <form onSubmit={handleRun} className="flex flex-wrap items-end gap-4 bg-card border border-border rounded-lg p-4">
        <div>
          <label className="block text-xs font-medium text-muted-foreground mb-1">Method</label>
          <Select value={method} onValueChange={(v) => setMethod(v as AccountingMethod)}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="fifo">FIFO</SelectItem>
              <SelectItem value="avgcost">Average Cost</SelectItem>
              <SelectItem value="lifo">LIFO</SelectItem>
              <SelectItem value="hifo">HIFO</SelectItem>
              <SelectItem value="specificid">Specific ID</SelectItem>
              <SelectItem value="section104">Section 104 (UK)</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <label className="block text-xs font-medium text-muted-foreground mb-1">Currency</label>
          <Select value={currency} onValueChange={setCurrency}>
            <SelectTrigger className="w-24">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="EUR">EUR</SelectItem>
              <SelectItem value="USD">USD</SelectItem>
              <SelectItem value="GBP">GBP</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button type="submit" disabled={running}>
          {running ? 'Running…' : 'Run Accounting'}
        </Button>
        {error && <p className="text-sm text-destructive">{error}</p>}
      </form>

      {summary && (
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'Total Cost', value: fmtCents(summary.total_cost_fiat, currency) },
            { label: 'Unrealised Gain', value: fmtCents(summary.unrealised_gain_fiat, currency) },
            { label: 'Realised Gain', value: fmtCents(summary.realised_gain_fiat, currency) },
          ].map(({ label, value }) => (
            <div key={label} className="bg-card border border-border rounded-lg p-4">
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="text-xl font-semibold mt-1">{value}</p>
            </div>
          ))}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        <ExportBar walletID={walletID} report="pnl" label="P&L:" />
        <ExportBar walletID={walletID} report="balance-sheet" label="Balance Sheet:" />
      </div>

      {records.length > 0 && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-secondary/60 border-b border-border">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">UTXO</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Acquired</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Disposed</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Cost</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Proceeds</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Gain</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {records.map((r) => (
                <tr key={r.id} className="hover:bg-secondary">
                  <td className="px-4 py-3 font-mono text-xs text-foreground">
                    {r.txid.slice(0, 10)}…:{r.vout}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {r.acquired_at ? new Date(r.acquired_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {r.disposed_at ? new Date(r.disposed_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">{fmtCents(r.cost_fiat, currency)}</td>
                  <td className="px-4 py-3 text-right">{fmtCents(r.proceeds_fiat ?? null, currency)}</td>
                  <td className={cn('px-4 py-3 text-right font-medium',
                    r.gain_fiat != null && r.gain_fiat < 0 ? 'text-destructive' : 'text-success'
                  )}>
                    {fmtCents(r.gain_fiat ?? null, currency)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
