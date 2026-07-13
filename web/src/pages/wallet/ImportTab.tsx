import { useEffect, useRef, useState } from 'react'
import {
  getImportJob,
  importWallet,
  listImportJobs,
  type ImportJob,
} from '../../api/wallets'
import { cn } from '../../lib/utils'
import { usePoll } from '../../hooks/usePoll'

export function ImportTab({ walletID }: { walletID: string }) {
  const [jobs, setJobs] = useState<ImportJob[]>([])
  const [uploading, setUploading] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [loadError, setLoadError] = useState('')
  const [dragOver, setDragOver] = useState(false)
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

  async function upload(file: File) {
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

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) upload(file)
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) upload(file)
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(true)
  }

  function handleDragLeave(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
  }

  return (
    <div className="space-y-6">
      {loadError && (
        <p role="alert" className="text-sm text-destructive">{loadError}</p>
      )}

      <div className="bg-card border border-border rounded-lg p-4 space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Upload wallet export</label>
          <p className="text-xs text-muted-foreground mb-3">
            Supported formats: Sparrow (JSON, CSV, BIP329 .jsonl) · Nunchuk (JSON, BSMS, BIP329 .jsonl) ·
            Coldcard (coldcard-export.json) · Specter Desktop (JSON descriptor export) ·
            Electrum (JSON wallet export, unencrypted only) · Generic JSON with descriptor field (Jade, Passport, SeedSigner, etc.)
          </p>

          {/* Drop zone */}
          <div
            role="button"
            tabIndex={uploading ? -1 : 0}
            aria-label="Upload wallet export file"
            aria-disabled={uploading}
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onClick={() => !uploading && fileRef.current?.click()}
            onKeyDown={(e) => {
              if ((e.key === 'Enter' || e.key === ' ') && !uploading) {
                e.preventDefault()
                fileRef.current?.click()
              }
            }}
            className={cn(
              'flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-6 py-10 text-center cursor-pointer transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-ring',
              dragOver
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-primary/50',
              uploading && 'pointer-events-none opacity-60',
            )}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
            </svg>
            <span className="text-sm text-muted-foreground">
              {dragOver ? 'Drop to import' : 'Drag & drop a file here, or click to browse'}
            </span>
          </div>
          <input
            ref={fileRef}
            type="file"
            aria-label="Upload wallet export file"
            accept=".json,.csv,.bsms,.jsonl"
            className="hidden"
            onChange={handleFileChange}
            disabled={uploading}
          />
        </div>

        {status && <p className="text-sm text-muted-foreground">{status}</p>}
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>

      {jobs.length > 0 && (
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-secondary/60 border-b border-border">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">File</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Source</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Status</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Records</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {jobs.map((j) => (
                <tr key={j.id}>
                  <td className="px-4 py-3 text-foreground">{j.filename || '—'}</td>
                  <td className="px-4 py-3 text-muted-foreground capitalize">{j.source}</td>
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
                  <td className="px-4 py-3 text-right text-muted-foreground">{j.records_imported ?? '—'}</td>
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
