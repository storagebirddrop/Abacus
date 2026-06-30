import { useEffect, useState } from 'react'
import { startSync, getSyncJob, listSyncJobs, type SyncJob } from '../../api/sync'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { usePoll } from '../../hooks/usePoll'

export function SyncPanel({ walletID }: { walletID: string }) {
  const [jobs, setJobs] = useState<SyncJob[]>([])
  const [syncing, setSyncing] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const poll = usePoll()

  useEffect(() => {
    listSyncJobs(walletID).then(setJobs).catch(() => {})
  }, [walletID])

  async function handleSync() {
    setSyncing(true)
    setError('')
    setStatus('Starting sync…')
    try {
      const { job_id } = await startSync(walletID)
      setStatus(`Sync job started (${job_id.slice(0, 8)}…)`)
      poll.start(async () => {
        const updated = await getSyncJob(job_id)
        setJobs((prev) => {
          const exists = prev.find((j) => j.id === updated.id)
          return exists ? prev.map((j) => (j.id === updated.id ? updated : j)) : [updated, ...prev]
        })
        if (updated.status === 'done' || updated.status === 'failed') {
          poll.stop()
          setStatus(updated.status === 'done'
            ? `Done — ${updated.tx_found} transactions, ${updated.addresses_scanned} addresses scanned`
            : `Failed: ${updated.error_message}`)
          setSyncing(false)
        }
      }, 2000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Sync failed')
      setSyncing(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4 space-y-4">
        <div>
          <h3 className="text-sm font-medium mb-1">Blockchain Sync</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Derives addresses from the wallet descriptor and fetches transaction history from the configured blockchain backend.
            Requires the wallet to have a descriptor set.
          </p>
        </div>
        {status && <p className="text-sm text-slate-600 dark:text-slate-300">{status}</p>}
        {error && <p className="text-sm text-red-500">{error}</p>}
        <Button onClick={handleSync} disabled={syncing}>
          {syncing ? 'Syncing…' : 'Sync from Blockchain'}
        </Button>
      </div>

      {jobs.length > 0 && (
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Backend</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Status</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Addresses</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Transactions</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {jobs.map((j) => (
                <tr key={j.id}>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400 capitalize">{j.backend}</td>
                  <td className="px-4 py-3">
                    <span className={cn(
                      'text-xs px-2 py-0.5 rounded-full',
                      j.status === 'done' && 'bg-green-100 text-green-700',
                      j.status === 'failed' && 'bg-red-100 text-red-700',
                      j.status === 'running' && 'bg-blue-100 text-blue-700',
                      j.status === 'pending' && 'bg-slate-100 text-slate-600 dark:text-slate-300',
                    )}>
                      {j.status}
                    </span>
                    {j.error_message && (
                      <span className="ml-2 text-xs text-red-500">{j.error_message}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-500 dark:text-slate-400">{j.addresses_scanned}</td>
                  <td className="px-4 py-3 text-right text-slate-500 dark:text-slate-400">{j.tx_found}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">
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
