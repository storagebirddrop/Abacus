import { useEffect, useState } from 'react'
import {
  getAccountingSummary,
  listCostBasis,
  runAccounting,
  type AccountingSummary,
  type CostBasisRecord,
} from '../../api/accounting'
import { Button } from '../../components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../components/ui/select'
import { cn } from '../../lib/utils'
import { ExportBar } from './ExportBar'

function fmtCents(cents: number | null) {
  if (cents === null || cents === 0) return '—'
  return `€${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}`
}

export function AccountingTab({ walletID }: { walletID: string }) {
  const [method, setMethod] = useState<'fifo' | 'avgcost'>('fifo')
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
        <p role="alert" className="text-sm text-red-500">{loadError}</p>
      )}
      <form onSubmit={handleRun} className="flex items-end gap-4 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4">
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Method</label>
          <Select value={method} onValueChange={(v) => setMethod(v as 'fifo' | 'avgcost')}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="fifo">FIFO</SelectItem>
              <SelectItem value="avgcost">Average Cost</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 mb-1">Currency</label>
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
        {error && <p className="text-sm text-red-500">{error}</p>}
      </form>

      {summary && (
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'Total Cost', value: fmtCents(summary.total_cost_fiat) },
            { label: 'Unrealised Gain', value: fmtCents(summary.unrealised_gain_fiat) },
            { label: 'Realised Gain', value: fmtCents(summary.realised_gain_fiat) },
          ].map(({ label, value }) => (
            <div key={label} className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4">
              <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
              <p className="text-xl font-semibold mt-1">{value}</p>
            </div>
          ))}
        </div>
      )}

      <div className="flex gap-2 mt-2">
        <ExportBar walletID={walletID} report="pnl" />
        <ExportBar walletID={walletID} report="balance-sheet" />
      </div>
      {records.length > 0 && (
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">UTXO</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Acquired</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Disposed</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Cost</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Proceeds</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Gain</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {records.map((r) => (
                <tr key={r.id} className="hover:bg-slate-50 dark:hover:bg-slate-800">
                  <td className="px-4 py-3 font-mono text-xs text-slate-700 dark:text-slate-200">
                    {r.txid.slice(0, 10)}…:{r.vout}
                  </td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">
                    {r.acquired_at ? new Date(r.acquired_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">
                    {r.disposed_at ? new Date(r.disposed_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">{fmtCents(r.cost_fiat)}</td>
                  <td className="px-4 py-3 text-right">{fmtCents(r.proceeds_fiat ?? null)}</td>
                  <td className={cn('px-4 py-3 text-right font-medium', r.gain_fiat != null && r.gain_fiat < 0 ? 'text-red-500' : 'text-green-600')}>
                    {fmtCents(r.gain_fiat ?? null)}
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
