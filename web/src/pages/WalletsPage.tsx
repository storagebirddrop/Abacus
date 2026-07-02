import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createWallet, deleteWallet, importWallet, listWallets, type Wallet } from '../api/wallets'
import { Button } from '../components/ui/button'
import { useToast } from '../components/Toast'
import { useConfirm } from '../components/ConfirmDialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../components/ui/dialog'
import { cn } from '../lib/utils'

function AddWalletDialog({ onCreated }: { onCreated: () => void }) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [descriptor, setDescriptor] = useState('')
  const [showDescriptor, setShowDescriptor] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  function reset() {
    setName('')
    setFile(null)
    setDescriptor('')
    setShowDescriptor(false)
    setError('')
  }

  function handleFileSelect(f: File) {
    setFile(f)
    if (!name) {
      // Auto-fill name from filename (strip extension)
      setName(f.name.replace(/\.[^.]+$/, ''))
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (f) handleFileSelect(f)
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const f = e.dataTransfer.files[0]
    if (f) handleFileSelect(f)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) {
      setError('Wallet name is required.')
      return
    }
    setLoading(true)
    setError('')
    try {
      const wallet = await createWallet({ name: name.trim(), descriptor })
      if (file) {
        await importWallet(wallet.id, file)
        setOpen(false)
        reset()
        navigate(`/wallets/${wallet.id}`)
      } else {
        setOpen(false)
        reset()
        onCreated()
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { setOpen(v); if (!v) reset() }}>
      <DialogTrigger asChild>
        <Button>Add Wallet</Button>
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Wallet</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* File drop zone */}
          <div>
            <label className="block text-sm font-medium mb-1">Import file <span className="text-slate-400 font-normal">(optional)</span></label>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
              Sparrow, Nunchuk, Coldcard, Specter, Electrum, or any JSON/BSMS/JSONL export.
            </p>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={(e) => { e.preventDefault(); setDragOver(false) }}
              onClick={() => !loading && fileRef.current?.click()}
              className={cn(
                'flex flex-col items-center justify-center gap-1.5 rounded-lg border-2 border-dashed px-4 py-6 text-center cursor-pointer transition-colors',
                dragOver
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/30'
                  : file
                  ? 'border-green-400 bg-green-50 dark:bg-green-950/20'
                  : 'border-slate-300 dark:border-slate-700 hover:border-slate-400 dark:hover:border-slate-500',
                loading && 'pointer-events-none opacity-60',
              )}
            >
              {file ? (
                <>
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span className="text-sm text-green-700 dark:text-green-400 font-medium">{file.name}</span>
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setFile(null) }}
                    className="text-xs text-slate-400 hover:text-slate-600 underline"
                  >
                    Remove
                  </button>
                </>
              ) : (
                <>
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-slate-400 dark:text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
                  </svg>
                  <span className="text-sm text-slate-600 dark:text-slate-300">
                    {dragOver ? 'Drop to import' : 'Drag & drop a file, or click to browse'}
                  </span>
                </>
              )}
              <input
                ref={fileRef}
                type="file"
                accept=".json,.csv,.bsms,.jsonl"
                className="hidden"
                onChange={handleFileChange}
                disabled={loading}
              />
            </div>
          </div>

          {/* Name */}
          <div>
            <label className="block text-sm font-medium mb-1">Wallet name</label>
            <input
              className="w-full border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900 bg-white dark:bg-slate-900"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Bitcoin Wallet"
              required
            />
          </div>

          {/* Advanced: descriptor */}
          <div>
            <button
              type="button"
              onClick={() => setShowDescriptor((v) => !v)}
              className="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 underline"
            >
              {showDescriptor ? 'Hide descriptor' : 'Enter output descriptor (advanced)'}
            </button>
            {showDescriptor && (
              <textarea
                className="mt-2 w-full border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-slate-900 bg-white dark:bg-slate-900 resize-none"
                value={descriptor}
                onChange={(e) => setDescriptor(e.target.value)}
                placeholder="wpkh([fingerprint/path]xpub…)"
                rows={3}
              />
            )}
          </div>

          {error && <p className="text-sm text-red-500">{error}</p>}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? (file ? 'Creating & importing…' : 'Creating…') : (file ? 'Add & Import' : 'Add Wallet')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function WalletsPage() {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const { toast } = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<'name' | 'created'>('name')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')

  function toggleSort(key: 'name' | 'created') {
    if (key === sortKey) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  const visible = wallets
    .filter((w) => {
      const q = search.trim().toLowerCase()
      if (!q) return true
      return w.name.toLowerCase().includes(q) || w.fingerprint.toLowerCase().includes(q)
    })
    .sort((a, b) => {
      const cmp =
        sortKey === 'name'
          ? a.name.localeCompare(b.name)
          : (a.created_at || '').localeCompare(b.created_at || '')
      return sortDir === 'asc' ? cmp : -cmp
    })

  async function load() {
    try {
      const data = await listWallets()
      setWallets(data ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load wallets')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  async function handleDelete(id: string, name: string) {
    const ok = await confirm({
      title: 'Delete wallet',
      description: `Delete "${name}"? This removes all transactions, ledger entries, and accounting data. This cannot be undone.`,
      confirmText: 'Delete',
      destructive: true,
    })
    if (!ok) return
    try {
      await deleteWallet(id)
      setWallets((prev) => prev.filter((w) => w.id !== id))
      toast(`Deleted "${name}"`, 'success')
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Delete failed', 'error')
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Wallets</h1>
        <AddWalletDialog onCreated={load} />
      </div>

      {loading && <p className="text-slate-500 dark:text-slate-400">Loading…</p>}
      {error && <p className="text-red-500">{error}</p>}

      {!loading && wallets.length === 0 && (
        <div className="text-center py-16 text-slate-400">
          <p className="text-lg">No wallets yet</p>
          <p className="text-sm mt-1">Add a wallet by importing a Sparrow, Nunchuk, Coldcard, or other export file.</p>
        </div>
      )}

      {wallets.length > 0 && (
        <>
          <input
            type="search"
            aria-label="Search wallets"
            placeholder="Search by name or fingerprint…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full sm:w-72 mb-4 border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-slate-400 bg-white dark:bg-slate-900"
          />

          {visible.length === 0 ? (
            <div className="text-center py-12 text-slate-400">
              <p>No wallets match "{search}".</p>
            </div>
          ) : (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">
                    <button className="hover:text-slate-900" onClick={() => toggleSort('name')}>
                      Name{sortKey === 'name' ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
                    </button>
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Type</th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">Network</th>
                  <th className="text-left px-4 py-3 font-medium text-slate-600 dark:text-slate-300">
                    <button className="hover:text-slate-900" onClick={() => toggleSort('created')}>
                      Created{sortKey === 'created' ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
                    </button>
                  </th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {visible.map((w) => (
                <tr key={w.id} className="hover:bg-slate-50 dark:hover:bg-slate-800">
                  <td className="px-4 py-3">
                    <Link to={`/wallets/${w.id}`} className="font-medium text-slate-900 dark:text-slate-100 hover:underline">
                      {w.name}
                    </Link>
                    {w.fingerprint && (
                      <span className="ml-2 text-xs text-slate-400 font-mono">{w.fingerprint}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400 capitalize">{w.type || '—'}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400 capitalize">{w.network || '—'}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">
                    {w.created_at ? new Date(w.created_at).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label={`Delete wallet ${w.name}`}
                      onClick={() => handleDelete(w.id, w.name)}
                      className="text-red-500 hover:text-red-600 hover:bg-red-50"
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
              </tbody>
            </table>
          </div>
          )}
        </>
      )}
    </div>
  )
}
