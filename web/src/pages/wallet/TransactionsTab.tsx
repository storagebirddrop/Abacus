import { useEffect, useState } from 'react'
import { listTransactions, type Transaction } from '../../api/wallets'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { ExportBar } from './ExportBar'

type SortKey = 'date' | 'fee'
type StatusFilter = '' | 'confirmed' | 'pending'

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

  // Debounce the search box so typing doesn't fire a request per keystroke.
  const [debouncedSearch, setDebouncedSearch] = useState('')
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(t)
  }, [search])

  // Any filter/sort change resets to the first page.
  useEffect(() => {
    setPage(1)
  }, [debouncedSearch, status, sort, dir])

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

  const offset = (page - 1) * limit
  const arrow = (key: SortKey) => (sort === key ? (dir === 'asc' ? ' ↑' : ' ↓') : '')

  return (
    <div>
      <ExportBar walletID={walletID} report="transactions" />

      <div className="flex flex-wrap items-center gap-3 mb-4">
        <input
          type="search"
          aria-label="Search transactions by txid"
          placeholder="Search by txid…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full sm:w-72 border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400 bg-white dark:bg-slate-900"
        />
        <select
          aria-label="Filter by status"
          value={status}
          onChange={(e) => setStatus(e.target.value as StatusFilter)}
          className="border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400 bg-white dark:bg-slate-900"
        >
          <option value="">All statuses</option>
          <option value="confirmed">Confirmed</option>
          <option value="pending">Pending</option>
        </select>
      </div>

      {loading ? (
        <p className="text-slate-500 dark:text-slate-400 p-6">Loading…</p>
      ) : error ? (
        <p className="text-red-500 p-6">{error}</p>
      ) : txs.length === 0 ? (
        <p className="text-slate-400 p-6">
          {debouncedSearch || status ? 'No transactions match your filters.' : 'No transactions. Import a wallet file first.'}
        </p>
      ) : (
        <>
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">
                    <button className="hover:text-slate-900 dark:hover:text-slate-100" onClick={() => toggleSort('date')}>
                      Date{arrow('date')}
                    </button>
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Txid</th>
                  <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">
                    <button className="hover:text-slate-900 dark:hover:text-slate-100" onClick={() => toggleSort('fee')}>
                      Fee (sats){arrow('fee')}
                    </button>
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {txs.map((tx) => (
                  <tr key={tx.id} className="hover:bg-slate-50 dark:hover:bg-slate-800">
                    <td className="px-4 py-3 text-slate-500 dark:text-slate-400">
                      {tx.block_time ? new Date(tx.block_time).toLocaleDateString() : 'Unconfirmed'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-700 dark:text-slate-200">
                      {tx.txid.slice(0, 16)}…{tx.txid.slice(-8)}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-500 dark:text-slate-400">{tx.fee_sats ?? '—'}</td>
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
          <div className="flex items-center gap-4 mt-4 text-sm text-slate-500 dark:text-slate-400">
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
