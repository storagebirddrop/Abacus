import { useEffect, useRef, useState } from 'react'
import {
  getImportJob,
  importWallet,
  listImportJobs,
  type ImportJob,
} from '../../api/wallets'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { usePoll } from '../../hooks/usePoll'

export function ImportTab({ walletID }: { walletID: string }) {
  const [jobs, setJobs] = useState<ImportJob[]>([])
  const [uploading, setUploading] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [loadError, setLoadError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const poll = usePoll()

  useEffect(() => {
    setLoadError('')
    listImportJobs(walletID)
      .then((j) => setJobs(j ?? []))
      .catch((err: unknown) =>
        setLoadError(err instanceof Error ? err.message : 'Failed to load import history'),
      )
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
      poll.start(async () => {
        const updated = await getImportJob(job.id)
        setJobs((prev) => prev.map((j) => (j.id === updated.id ? updated : j)))
        if (updated.status === 'done' || updated.status === 'failed') {
          poll.stop()
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
      {loadError && (
        <p role="alert" className="text-sm text-red-500">{loadError}</p>
      )}
      <form onSubmit={handleUpload} className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4 space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Upload wallet export</label>
          <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
            Supported formats: Sparrow (JSON, CSV, BIP329 .jsonl) · Nunchuk (JSON, BSMS, BIP329 .jsonl) ·
            Coldcard (coldcard-export.json) · Specter Desktop (JSON descriptor export) ·
            Electrum (JSON wallet export, unencrypted only) · Generic JSON with descriptor field (Jade, Passport, SeedSigner, etc.)
          </p>
          <input
            ref={fileRef}
            type="file"
            accept=".json,.csv,.bsms,.jsonl"
            className="text-sm"
            required
          />
        </div>
        {status && <p className="text-sm text-slate-600 dark:text-slate-300">{status}</p>}
        {error && <p className="text-sm text-red-500">{error}</p>}
        <Button type="submit" disabled={uploading}>
          {uploading ? 'Importing…' : 'Import'}
        </Button>
      </form>

      {jobs.length > 0 && (
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">File</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Source</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Status</th>
                <th className="text-right px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Records</th>
                <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {jobs.map((j) => (
                <tr key={j.id}>
                  <td className="px-4 py-3 text-slate-700 dark:text-slate-200">{j.filename || '—'}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400 capitalize">{j.source}</td>
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
                  <td className="px-4 py-3 text-right text-slate-500 dark:text-slate-400">{j.records_imported ?? '—'}</td>
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
