import { useEffect, useState } from 'react'
import { listTransactions, patchTransaction, type Transaction } from '../../api/wallets'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { ExportBar } from './ExportBar'
import { ImportModal } from './ImportModal'

type SortKey = 'date' | 'fee'
type StatusFilter = '' | 'confirmed' | 'pending'

const CATEGORIES = [
  'unknown', 'income', 'expense', 'transfer', 'exchange',
  'mining', 'donation', 'salary', 'gift', 'coinjoin',
  'lightning', 'correction', 'fee',
] as const

const CATEGORY_STYLES: Record<string, string> = {
  income:     'bg-green-900/40 text-success light:bg-green-100',
  mining:     'bg-green-900/40 text-success light:bg-green-100',
  salary:     'bg-green-900/40 text-success light:bg-green-100',
  donation:   'bg-green-900/40 text-success light:bg-green-100',
  gift:       'bg-green-900/40 text-success light:bg-green-100',
  expense:    'bg-red-900/40 text-destructive light:bg-red-100',
  fee:        'bg-red-900/40 text-destructive light:bg-red-100',
  transfer:   'bg-blue-900/40 text-blue-400 light:bg-blue-100 light:text-blue-700',
  exchange:   'bg-purple-900/40 text-purple-400 light:bg-purple-100 light:text-purple-700',
  coinjoin:   'bg-orange-900/40 text-orange-400 light:bg-orange-100 light:text-orange-700',
  lightning:  'bg-yellow-900/40 text-yellow-400 light:bg-yellow-100 light:text-yellow-700',
  correction: 'bg-secondary text-muted-foreground',
  unknown:    'bg-secondary text-muted-foreground',
}

function fmtSats(sats: number): string {
  if (sats === 0) return '—'
  const btc = Math.abs(sats) / 1e8
  const formatted = btc.toFixed(8).replace(/\.?0+$/, '')
  return (sats > 0 ? '+' : '−') + formatted + ' BTC'
}

function CategoryCell({
  tx,
  walletID,
  onUpdate,
}: {
  tx: Transaction
  walletID: string
  onUpdate: (txid: string, category: string) => void
}) {
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)

  async function handleChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const newCat = e.target.value
    setSaving(true)
    try {
      await patchTransaction(walletID, tx.txid, { category: newCat })
      onUpdate(tx.txid, newCat)
    } finally {
      setSaving(false)
      setEditing(false)
    }
  }

  if (editing) {
    return (
      <select
        autoFocus
        value={tx.category}
        onChange={handleChange}
        onBlur={() => setEditing(false)}
        disabled={saving}
        className="text-xs border border-border rounded px-1 py-0.5 bg-card focus:outline-none focus:ring-1 focus:ring-ring"
      >
        {CATEGORIES.map((c) => (
          <option key={c} value={c}>{c}</option>
        ))}
      </select>
    )
  }

  return (
    <button
      onClick={() => setEditing(true)}
      title="Click to change category"
      className={cn(
        'text-xs px-2 py-0.5 rounded-full cursor-pointer transition-opacity hover:opacity-75',
        CATEGORY_STYLES[tx.category] ?? CATEGORY_STYLES.unknown,
      )}
    >
      {tx.category || 'unknown'}
    </button>
  )
}

export function TransactionsTab({ walletID }: { walletID: string }) {
  const [txs, setTxs] = useState<Transaction[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<StatusFilter>('')
  const [sort, setSort] = useState<SortKey>('date')
  const [dir, setDir] = useState<'asc' | 'desc'>('desc')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const limit = 50

  const [debouncedSearch, setDebouncedSearch] = useState('')
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(t)
  }, [search])

  useEffect(() => { setPage(1) }, [debouncedSearch, status, sort, dir])

  useEffect(() => {
    setLoading(true)
    setError('')
    listTransactions(walletID, { page, limit, search: debouncedSearch, status, sort, dir })
      .then((data) => {
        setTxs(data.data ?? [])
        setTotal(data.total ?? 0)
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed'))
      .finally(() => setLoading(false))
  }, [walletID, page, debouncedSearch, status, sort, dir])

  function toggleSort(key: SortKey) {
    if (key === sort) {
      setDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSort(key)
      setDir('desc')
    }
  }

  function handleCategoryUpdate(txid: string, category: string) {
    setTxs((prev) => prev.map((t) => t.txid === txid ? { ...t, category } : t))
  }

  const offset = (page - 1) * limit
  const arrow = (key: SortKey) => (sort === key ? (dir === 'asc' ? ' ↑' : ' ↓') : '')

  return (
    <div>
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <ExportBar walletID={walletID} report="transactions" />
        <div className="ml-auto">
          <ImportModal walletID={walletID} />
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-4">
        <input
          type="search"
          aria-label="Search transactions by txid"
          placeholder="Search by txid…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full sm:w-72 border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring bg-card"
        />
        <select
          aria-label="Filter by status"
          value={status}
          onChange={(e) => setStatus(e.target.value as StatusFilter)}
          className="border border-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring bg-card"
        >
          <option value="">All statuses</option>
          <option value="confirmed">Confirmed</option>
          <option value="pending">Pending</option>
        </select>
      </div>

      {loading ? (
        <p className="text-muted-foreground p-6">Loading…</p>
      ) : error ? (
        <p className="text-destructive p-6">{error}</p>
      ) : txs.length === 0 ? (
        <p className="text-muted-foreground p-6">
          {debouncedSearch || status
            ? 'No transactions match your filters.'
            : 'No transactions yet — import a wallet file to get started.'}
        </p>
      ) : (
        <>
          <div className="bg-card border border-border rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-secondary/60 border-b border-border">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">
                    <button className="hover:text-primary" onClick={() => toggleSort('date')}>
                      Date{arrow('date')}
                    </button>
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Txid</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">Amount</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Category</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">
                    <button className="hover:text-primary" onClick={() => toggleSort('fee')}>
                      Fee (sats){arrow('fee')}
                    </button>
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {txs.map((tx) => (
                  <tr key={tx.id} className="hover:bg-secondary">
                    <td className="px-4 py-3 text-muted-foreground">
                      {tx.block_time ? new Date(tx.block_time).toLocaleDateString() : 'Unconfirmed'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-foreground">
                      {tx.txid.slice(0, 16)}…{tx.txid.slice(-8)}
                    </td>
                    <td className={cn(
                      'px-4 py-3 text-right font-medium tabular-nums',
                      tx.net_sats > 0
                        ? 'text-success'
                        : tx.net_sats < 0
                        ? 'text-destructive'
                        : 'text-muted-foreground'
                    )}>
                      {fmtSats(tx.net_sats)}
                    </td>
                    <td className="px-4 py-3">
                      <CategoryCell tx={tx} walletID={walletID} onUpdate={handleCategoryUpdate} />
                    </td>
                    <td className="px-4 py-3 text-right text-muted-foreground">{tx.fee_sats ?? '—'}</td>
                    <td className="px-4 py-3">
                      <span className={cn(
                        'text-xs px-2 py-0.5 rounded-full',
                        tx.confirmed ? 'bg-green-100 text-success' : 'bg-yellow-100 text-yellow-700'
                      )}>
                        {tx.confirmed ? 'Confirmed' : 'Pending'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex items-center gap-4 mt-4 text-sm text-muted-foreground">
            <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
              Previous
            </Button>
            <span>Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}</span>
            <Button variant="outline" size="sm" disabled={offset + limit >= total} onClick={() => setPage((p) => p + 1)}>
              Next
            </Button>
          </div>
        </>
      )}
    </div>
  )
}
