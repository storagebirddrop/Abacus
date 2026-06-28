import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  getWallet,
  getImportJob,
  importWallet,
  listImportJobs,
  listTransactions,
  type ImportJob,
  type Transaction,
  type Wallet,
} from '../api/wallets'
import {
  getAccountingSummary,
  listCostBasis,
  runAccounting,
  type AccountingSummary,
  type CostBasisRecord,
} from '../api/accounting'
import { Button } from '../components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { cn } from '../lib/utils'

type Tab = 'transactions' | 'accounting' | 'import'

function fmtCents(cents: number | null) {
  if (cents === null || cents === 0) return '—'
  return `€${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })}`
}

function TransactionsTab({ walletID }: { walletID: string }) {
  const [txs, setTxs] = useState<Transaction[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const limit = 50

  useEffect(() => {
    setLoading(true)
    listTransactions(walletID, limit, offset)
      .then((data) => {
        setTxs(data.transactions ?? [])
        setTotal(data.total ?? 0)
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed'))
      .finally(() => setLoading(false))
  }, [walletID, offset])

  if (loading) return <p className="text-slate-500 p-6">Loading…</p>
  if (error) return <p className="text-red-500 p-6">{error}</p>

  return (
    <div>
      {txs.length === 0 ? (
        <p className="text-slate-400 p-6">No transactions. Import a wallet file first.</p>
      ) : (
        <>
          <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 border-b border-slate-200">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-slate-600">Date</th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600">Txid</th>
                  <th className="text-right px-4 py-3 font-medium text-slate-600">Fee (sats)</th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {txs.map((tx) => (
                  <tr key={tx.id} className="hover:bg-slate-50">
                    <td className="px-4 py-3 text-slate-500">
                      {tx.block_time ? new Date(tx.block_time).toLocaleDateString() : 'Unconfirmed'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-700">
                      {tx.txid.slice(0, 16)}…{tx.txid.slice(-8)}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-500">{tx.fee_sats ?? '—'}</td>
                    <td className="px-4 py-3">
                      <span className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        tx.confirmed ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'
                      )}>
                        {tx.confirmed ? 'Confirmed' : 'Pending'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex items-center gap-4 mt-4 text-sm text-slate-500">
            <Button variant="outline" size="sm" disabled={offset === 0} onClick={() => setOffset((o) => Math.max(0, o - limit))}>
              Previous
            </Button>
            <span>Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}</span>
            <Button variant="outline" size="sm" disabled={offset + limit >= total} onClick={() => setOffset((o) => o + limit)}>
              Next
            </Button>
          </div>
        </>
      )}
    </div>
  )
}

function AccountingTab({ walletID }: { walletID: string }) {
  const [method, setMethod] = useState<'fifo' | 'avgcost'>('fifo')
  const [currency, setCurrency] = useState('EUR')
  const [summary, setSummary] = useState<AccountingSummary | null>(null)
  const [records, setRecords] = useState<CostBasisRecord[]>([])
  const [running, setRunning] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    getAccountingSummary(walletID)
      .then(setSummary)
      .catch(() => {})
    listCostBasis(walletID)
      .then(setRecords)
      .catch(() => {})
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
      <form onSubmit={handleRun} className="flex items-end gap-4 bg-white border border-slate-200 rounded-lg p-4">
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">Method</label>
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
          <label className="block text-xs font-medium text-slate-600 mb-1">Currency</label>
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
            <div key={label} className="bg-white border border-slate-200 rounded-lg p-4">
              <p className="text-xs text-slate-500">{label}</p>
              <p className="text-xl font-semibold mt-1">{value}</p>
            </div>
          ))}
        </div>
      )}

      {records.length > 0 && (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600">UTXO</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Acquired</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Disposed</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600">Cost</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600">Proceeds</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600">Gain</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {records.map((r) => (
                <tr key={r.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-mono text-xs text-slate-700">
                    {r.txid.slice(0, 10)}…:{r.vout}
                  </td>
                  <td className="px-4 py-3 text-slate-500">
                    {r.acquired_at ? new Date(r.acquired_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-slate-500">
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

function ImportTab({ walletID }: { walletID: string }) {
  const [jobs, setJobs] = useState<ImportJob[]>([])
  const [uploading, setUploading] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    listImportJobs(walletID).then((j) => setJobs(j ?? [])).catch(() => {})
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
  }, [walletID])

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) return
    setUploading(true)
    setError('')
    setStatus('Uploading…')
    try {
      const job = await importWallet(walletID, file)
      setStatus(`Job started (${job.id.slice(0, 8)}…)`)
      setJobs((prev) => [job, ...prev])
      pollRef.current = setInterval(async () => {
        const updated = await getImportJob(job.id)
        setJobs((prev) => prev.map((j) => (j.id === updated.id ? updated : j)))
        if (updated.status === 'done' || updated.status === 'failed') {
          clearInterval(pollRef.current!)
          setStatus(updated.status === 'done'
            ? `Done — ${updated.records_imported} records imported`
            : `Failed: ${updated.error_message}`)
          setUploading(false)
        }
      }, 2000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Upload failed')
      setUploading(false)
    }
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleUpload} className="bg-white border border-slate-200 rounded-lg p-4 space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Upload wallet export</label>
          <p className="text-xs text-slate-500 mb-2">Supports Sparrow JSON, Nunchuk JSON, BSMS, BIP329 labels (.jsonl)</p>
          <input
            ref={fileRef}
            type="file"
            accept=".json,.csv,.bsms,.jsonl"
            className="text-sm"
            required
          />
        </div>
        {status && <p className="text-sm text-slate-600">{status}</p>}
        {error && <p className="text-sm text-red-500">{error}</p>}
        <Button type="submit" disabled={uploading}>
          {uploading ? 'Importing…' : 'Import'}
        </Button>
      </form>

      {jobs.length > 0 && (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600">File</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Source</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Status</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600">Records</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {jobs.map((j) => (
                <tr key={j.id}>
                  <td className="px-4 py-3 text-slate-700">{j.filename || '—'}</td>
                  <td className="px-4 py-3 text-slate-500 capitalize">{j.source}</td>
                  <td className="px-4 py-3">
                    <span className={cn(
                      'text-xs px-2 py-0.5 rounded-full',
                      j.status === 'done' && 'bg-green-100 text-green-700',
                      j.status === 'failed' && 'bg-red-100 text-red-700',
                      j.status === 'running' && 'bg-blue-100 text-blue-700',
                      j.status === 'pending' && 'bg-slate-100 text-slate-600',
                    )}>
                      {j.status}
                    </span>
                    {j.error_message && (
                      <span className="ml-2 text-xs text-red-500">{j.error_message}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-500">{j.records_imported ?? '—'}</td>
                  <td className="px-4 py-3 text-slate-500">
                    {j.started_at ? new Date(j.started_at).toLocaleString() : '—'}
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

export default function WalletPage() {
  const { id } = useParams<{ id: string }>()
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [tab, setTab] = useState<Tab>('transactions')

  useEffect(() => {
    if (id) getWallet(id).then(setWallet).catch(() => {})
  }, [id])

  if (!id) return null

  const tabs: { key: Tab; label: string }[] = [
    { key: 'transactions', label: 'Transactions' },
    { key: 'accounting', label: 'Accounting' },
    { key: 'import', label: 'Import' },
  ]

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{wallet?.name ?? 'Wallet'}</h1>
        {wallet?.fingerprint && (
          <p className="text-sm text-slate-500 font-mono mt-0.5">{wallet.fingerprint}</p>
        )}
      </div>

      <div className="flex gap-1 border-b border-slate-200 mb-6">
        {tabs.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={cn(
              'px-4 py-2 text-sm font-medium -mb-px border-b-2 transition-colors',
              tab === key
                ? 'border-slate-900 text-slate-900'
                : 'border-transparent text-slate-500 hover:text-slate-700'
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'transactions' && <TransactionsTab walletID={id} />}
      {tab === 'accounting' && <AccountingTab walletID={id} />}
      {tab === 'import' && <ImportTab walletID={id} />}
    </div>
  )
}
