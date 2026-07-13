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
  const [loadError, setLoadError] = useState('')
  const poll = usePoll()

  useEffect(() => {
    setLoadError('')
    listSyncJobs(walletID)
      .then((j) => setJobs(j ?? []))
      .catch((err: unknown) =>
        setLoadError(err instanceof Error ? err.message : 'Failed to load sync history'),
      )
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
      {loadError && (
        <p role="alert" className="text-sm text-destructive">{loadError}</p>
      )}
      <div className="bg-card border border-border rounded-lg p-4 space-y-4">
        <div>
          <h3 className="text-sm font-medium mb-1">Blockchain Sync</h3>
          <p className="text-xs text-muted-foreground">
            Derives addresses from the wallet descriptor and fetches transaction history from the configured blockchain backend.
            Requires the wallet to have a descriptor set.
          </p>
        </div>
        {status && <p className="text-sm text-muted-foreground">{status}</p>}
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button onClick={handleSync} disabled={syncing}>
          {syncing ? 'Syncing…' : 'Sync from Blockchain'}
        </Button>
      </div>

      {jobs.length > 0 && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-secondary/60 border-b border-border">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Backend</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Status</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Addresses</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Transactions</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {jobs.map((j) => (
                <tr key={j.id}>
                  <td className="px-4 py-3 text-muted-foreground capitalize">{j.backend}</td>
                  <td className="px-4 py-3">
                    <span className={cn(
                      'text-xs px-2 py-0.5 rounded-full',
                      j.status === 'done' && 'bg-green-100 text-success',
                      j.status === 'failed' && 'bg-red-100 text-destructive',
                      j.status === 'running' && 'bg-blue-100 text-blue-700',
                      j.status === 'pending' && 'bg-secondary text-muted-foreground',
                    )}>
                      {j.status}
                    </span>
                    {j.error_message && (
                      <span className="ml-2 text-xs text-destructive">{j.error_message}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">{j.addresses_scanned}</td>
                  <td className="px-4 py-3 text-right text-muted-foreground">{j.tx_found}</td>
                  <td className="px-4 py-3 text-muted-foreground">
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
