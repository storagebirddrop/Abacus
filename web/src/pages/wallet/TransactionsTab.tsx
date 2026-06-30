import { useEffect, useState } from 'react'
import { listTransactions, type Transaction } from '../../api/wallets'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { ExportBar } from './ExportBar'

export function TransactionsTab({ walletID }: { walletID: string }) {
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
      <ExportBar walletID={walletID} report="transactions" />
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
