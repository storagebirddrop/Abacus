import { useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createWallet, importWallet } from '../api/wallets'
import { Button } from './ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog'
import { cn } from '../lib/utils'

const EXCHANGES = ['Bitvavo', 'Bitonic', 'Kraken', 'Coinbase', 'Strike'] as const
type Exchange = (typeof EXCHANGES)[number]

export function AddWalletDialog({ onCreated }: { onCreated: () => void }) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [walletMode, setWalletMode] = useState<'onchain' | 'exchange'>('onchain')
  const [exchange, setExchange] = useState<Exchange>('Bitvavo')
  const [name, setName] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [descriptor, setDescriptor] = useState('')
  const [showDescriptor, setShowDescriptor] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  function reset() {
    setWalletMode('onchain')
    setExchange('Bitvavo')
    setName('')
    setFile(null)
    setDescriptor('')
    setShowDescriptor(false)
    setError('')
  }

  function handleFileSelect(f: File) {
    setFile(f)
    if (!name) {
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

  function handleModeChange(mode: 'onchain' | 'exchange') {
    setWalletMode(mode)
    setFile(null)
    setDescriptor('')
    setShowDescriptor(false)
    setError('')
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const walletName = name.trim() || (walletMode === 'exchange' ? exchange : '')
    if (!walletName) {
      setError('Wallet name is required.')
      return
    }
    setLoading(true)
    setError('')
    try {
      const source = walletMode === 'exchange' ? 'exchange' : 'manual'
      const desc = walletMode === 'exchange' ? `exchange:${exchange.toLowerCase()}` : descriptor
      const wallet = await createWallet({ name: walletName, descriptor: desc, source })
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

  const isExchange = walletMode === 'exchange'

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
          {/* Wallet type toggle */}
          <div>
            <label className="block text-sm font-medium mb-2">Wallet type</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => handleModeChange('onchain')}
                className={cn(
                  'flex-1 py-2 px-3 rounded-md text-sm font-medium border transition-colors',
                  !isExchange
                    ? 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900'
                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:border-slate-400',
                )}
              >
                On-chain wallet
              </button>
              <button
                type="button"
                onClick={() => handleModeChange('exchange')}
                className={cn(
                  'flex-1 py-2 px-3 rounded-md text-sm font-medium border transition-colors',
                  isExchange
                    ? 'bg-slate-900 text-white border-slate-900 dark:bg-slate-100 dark:text-slate-900'
                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:border-slate-400',
                )}
              >
                Exchange account
              </button>
            </div>
          </div>

          {/* Exchange selector (exchange mode only) */}
          {isExchange && (
            <div>
              <label className="block text-sm font-medium mb-1">Exchange</label>
              <select
                className="w-full border border-slate-200 dark:border-slate-800 rounded-md px-3 py-2 text-sm bg-white dark:bg-slate-900 focus:outline-none focus:ring-2 focus:ring-slate-900"
                value={exchange}
                onChange={(e) => setExchange(e.target.value as Exchange)}
              >
                {EXCHANGES.map((ex) => (
                  <option key={ex} value={ex}>{ex}</option>
                ))}
              </select>
            </div>
          )}

          {/* File drop zone */}
          <div>
            <label className="block text-sm font-medium mb-1">
              {isExchange ? 'Transaction export file' : 'Import file'}{' '}
              <span className="text-slate-400 font-normal">(optional)</span>
            </label>
            {isExchange ? (
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                Upload your {exchange} transaction history CSV export.
              </p>
            ) : (
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                Sparrow, Nunchuk, Coldcard, Specter, Electrum, or any JSON/BSMS/JSONL export.
              </p>
            )}
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
              placeholder={isExchange ? exchange : 'My Bitcoin Wallet'}
              required={!isExchange}
            />
          </div>

          {/* Advanced: descriptor (on-chain only) */}
          {!isExchange && (
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
          )}

          {error && <p className="text-sm text-red-500">{error}</p>}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading
                ? (file ? 'Creating & importing…' : 'Creating…')
                : (file ? 'Add & Import' : 'Add Wallet')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
